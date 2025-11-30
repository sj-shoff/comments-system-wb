package comments

import (
	"context"
	"fmt"

	"comments-system/internal/domain"

	"github.com/wb-go/wbf/zlog"
)

type commentsUsecase struct {
	repo   сommentsRepo
	cache  commentsCache
	logger *zlog.Zerolog
}

func NewCommentsUsecase(repo сommentsRepo, cache commentsCache, logger *zlog.Zerolog) *commentsUsecase {
	return &commentsUsecase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (u *commentsUsecase) CreateComment(ctx context.Context, comment domain.Comment) (domain.Comment, error) {
	if comment.Content == "" {
		return domain.Comment{}, ErrContentRequired
	}
	if comment.Author == "" {
		return domain.Comment{}, ErrAuthorRequired
	}
	if len(comment.Content) > 1000 {
		return domain.Comment{}, ErrContentTooLong
	}
	if len(comment.Author) > 50 {
		return domain.Comment{}, ErrAuthorTooLong
	}

	if comment.ParentID != nil {
		exists, err := u.repo.Exists(ctx, *comment.ParentID)
		if err != nil {
			return domain.Comment{}, err
		}
		if !exists {
			return domain.Comment{}, fmt.Errorf("%w: parent comment %d not found", ErrInvalidParentID, *comment.ParentID)
		}
	}

	createdComment, err := u.repo.Create(ctx, comment)
	if err != nil {
		return domain.Comment{}, err
	}

	if comment.ParentID != nil {
		if err := u.cache.InvalidateCommentTree(ctx, *comment.ParentID); err != nil {
			u.logger.Warn().Err(err).Int("parent_id", *comment.ParentID).Msg("Failed to invalidate parent comment tree cache")
		}
	}

	if err := u.cache.InvalidateComments(ctx, nil); err != nil {
		u.logger.Warn().Err(err).Msg("Failed to invalidate root comments cache")
	}

	return createdComment, nil
}

func (u *commentsUsecase) GetComments(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string) (domain.CommentTree, error) {

	if parentID != nil {
		cachedTree, err := u.cache.GetCommentTree(ctx, *parentID)
		if err == nil && cachedTree != nil {
			return domain.CommentTree{
				Comments: cachedTree,
				Total:    len(cachedTree),
				Page:     1,
				PageSize: len(cachedTree),
				HasNext:  false,
				HasPrev:  false,
			}, nil
		}

		tree, err := u.repo.GetTree(ctx, *parentID)
		if err != nil {
			return domain.CommentTree{}, err
		}

		if err := u.cache.SetCommentTree(ctx, *parentID, tree); err != nil {
			u.logger.Warn().Err(err).Int("parent_id", *parentID).Msg("Failed to cache comment tree")
		}

		return domain.CommentTree{
			Comments: tree,
			Total:    len(tree),
			Page:     1,
			PageSize: len(tree),
			HasNext:  false,
			HasPrev:  false,
		}, nil
	}

	cachedComments, totalCount, err := u.cache.GetComments(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder)
	if err == nil && cachedComments != nil {
		return domain.CommentTree{
			Comments: cachedComments,
			Total:    totalCount,
			Page:     page,
			PageSize: pageSize,
			HasNext:  (page * pageSize) < totalCount,
			HasPrev:  page > 1,
		}, nil
	}

	comments, totalCount, err := u.repo.GetByParent(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder)
	if err != nil {
		return domain.CommentTree{}, err
	}

	if err := u.cache.SetComments(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder, comments, totalCount); err != nil {
		u.logger.Warn().Err(err).Msg("Failed to cache comments")
	}

	return domain.CommentTree{
		Comments: comments,
		Total:    totalCount,
		Page:     page,
		PageSize: pageSize,
		HasNext:  (page * pageSize) < totalCount,
		HasPrev:  page > 1,
	}, nil
}

func (u *commentsUsecase) DeleteComment(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidCommentID
	}

	exists, err := u.repo.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCommentNotFound
	}

	var parentID *int
	comment, err := u.repo.GetTree(ctx, id)
	if err == nil && len(comment) > 0 {
		parentID = comment[0].ParentID
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := u.cache.InvalidateCommentTree(ctx, id); err != nil {
		u.logger.Warn().Err(err).Int("comment_id", id).Msg("Failed to invalidate comment tree cache")
	}

	if parentID != nil {
		if err := u.cache.InvalidateCommentTree(ctx, *parentID); err != nil {
			u.logger.Warn().Err(err).Int("parent_id", *parentID).Msg("Failed to invalidate parent comment tree cache")
		}
	}

	if err := u.cache.InvalidateComments(ctx, parentID); err != nil {
		u.logger.Warn().Err(err).Msg("Failed to invalidate comments cache")
	}

	return nil
}
