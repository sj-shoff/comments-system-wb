package comments

import (
	"context"
	"fmt"

	"comments-system/internal/domain"

	"github.com/wb-go/wbf/zlog"
)

type CommentsUsecase struct {
	repo   сommentsRepo
	cache  commentsCache
	logger *zlog.Zerolog
}

func NewCommentsUsecase(repo сommentsRepo, cache commentsCache, logger *zlog.Zerolog) *CommentsUsecase {
	return &CommentsUsecase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (u *CommentsUsecase) CreateComment(ctx context.Context, comment domain.Comment) (domain.Comment, error) {
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

func (u *CommentsUsecase) GetComments(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string) (domain.CommentTree, error) {
	if parentID != nil {
		parent, err := u.repo.GetByID(ctx, *parentID)
		if err != nil {
			return domain.CommentTree{}, err
		}

		children, err := u.cache.GetCommentTree(ctx, *parentID)
		if err != nil {
			children, err = u.repo.GetTree(ctx, *parentID)
			if err != nil {
				return domain.CommentTree{}, err
			}
			if err := u.cache.SetCommentTree(ctx, *parentID, children); err != nil {
				u.logger.Warn().Err(err).Int("parent_id", *parentID).Msg("Failed to cache comment tree")
			}
		}

		parent.Children = children
		return domain.CommentTree{
			Comments: []domain.Comment{parent},
			Total:    1,
			Page:     1,
			PageSize: 1,
			HasNext:  false,
			HasPrev:  false,
		}, nil
	}

	comments, totalCount, err := u.cache.GetComments(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder)
	if err != nil {
		u.logger.Warn().Err(err).Msg("Cache miss for comments, querying DB")
		comments, totalCount, err = u.repo.GetByParent(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder)
		if err != nil {
			return domain.CommentTree{}, err
		}

		if searchQuery == "" {
			for i := range comments {
				childComments, childErr := u.cache.GetCommentTree(ctx, comments[i].ID)
				if childErr != nil {
					childComments, childErr = u.repo.GetTree(ctx, comments[i].ID)
					if childErr != nil {
						u.logger.Warn().Err(childErr).Int("comment_id", comments[i].ID).Msg("Failed to get subtree")
						continue
					}
					if err := u.cache.SetCommentTree(ctx, comments[i].ID, childComments); err != nil {
						u.logger.Warn().Err(err).Int("comment_id", comments[i].ID).Msg("Failed to cache subtree")
					}
				}
				comments[i].Children = childComments
			}
		}

		if err := u.cache.SetComments(ctx, parentID, page, pageSize, searchQuery, sortBy, sortOrder, comments, totalCount); err != nil {
			u.logger.Warn().Err(err).Msg("Failed to set comments in cache")
		}
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

func (u *CommentsUsecase) DeleteComment(ctx context.Context, id int) error {
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

	parent, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = u.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if err := u.cache.InvalidateCommentTree(ctx, id); err != nil {
		u.logger.Warn().Err(err).Int("comment_id", id).Msg("Failed to invalidate comment tree cache")
	}

	if parent.ParentID != nil {
		if err := u.cache.InvalidateCommentTree(ctx, *parent.ParentID); err != nil {
			u.logger.Warn().Err(err).Int("parent_id", *parent.ParentID).Msg("Failed to invalidate parent comment tree cache")
		}
	}

	if err := u.cache.InvalidateComments(ctx, parent.ParentID); err != nil {
		u.logger.Warn().Err(err).Msg("Failed to invalidate comments cache")
	}

	return nil
}
