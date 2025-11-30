package posts

import (
	"context"
	"time"

	"comments-system/internal/domain"

	"github.com/wb-go/wbf/zlog"
)

type postsUsecase struct {
	repo   postsRepo
	cache  postsCache
	logger *zlog.Zerolog
}

func NewPostsUsecase(repo postsRepo, cache postsCache, logger *zlog.Zerolog) *postsUsecase {
	return &postsUsecase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (u *postsUsecase) CreatePost(ctx context.Context, post domain.Post) (domain.Post, error) {
	if post.Title == "" {
		return domain.Post{}, ErrTitleRequired
	}
	if post.Content == "" {
		return domain.Post{}, ErrContentRequired
	}
	if post.Author == "" {
		return domain.Post{}, ErrAuthorRequired
	}
	if len(post.Title) > 200 {
		return domain.Post{}, ErrTitleTooLong
	}
	if len(post.Content) > 10000 {
		return domain.Post{}, ErrContentTooLong
	}
	if len(post.Author) > 50 {
		return domain.Post{}, ErrAuthorTooLong
	}

	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	createdPost, err := u.repo.Create(ctx, post)
	if err != nil {
		return domain.Post{}, err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		u.cache.SetPost(ctx, createdPost)
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		u.cache.InvalidatePosts(ctx)
	}()

	return createdPost, nil
}

func (u *postsUsecase) GetPosts(ctx context.Context, page, pageSize int, searchQuery, sortBy, sortOrder string) (domain.PostTree, error) {
	cachedPosts, totalCount, err := u.cache.GetPosts(ctx, page, pageSize, searchQuery, sortBy, sortOrder)
	if err == nil {
		return domain.PostTree{
			Posts:    cachedPosts,
			Total:    totalCount,
			Page:     page,
			PageSize: pageSize,
			HasNext:  (page * pageSize) < totalCount,
			HasPrev:  page > 1,
		}, nil
	}

	if err != nil {
		u.logger.Warn().Err(err).Msg("Cache error, falling back to DB")
	}

	posts, totalCount, err := u.repo.GetAll(ctx, page, pageSize, searchQuery, sortBy, sortOrder)
	if err != nil {
		return domain.PostTree{}, err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		u.cache.SetPosts(ctx, page, pageSize, searchQuery, sortBy, sortOrder, posts, totalCount)
	}()

	return domain.PostTree{
		Posts:    posts,
		Total:    totalCount,
		Page:     page,
		PageSize: pageSize,
		HasNext:  (page * pageSize) < totalCount,
		HasPrev:  page > 1,
	}, nil
}

func (u *postsUsecase) GetPostByID(ctx context.Context, id int) (domain.Post, error) {
	cachedPost, err := u.cache.GetPost(ctx, id)
	if err == nil {
		return cachedPost, nil
	}

	if err != nil {
		u.logger.Warn().Err(err).Int("post_id", id).Msg("Cache error, falling back to DB")
	}

	post, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Post{}, err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		u.cache.SetPost(ctx, post)
	}()

	return post, nil
}

func (u *postsUsecase) DeletePost(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidPostID
	}

	exists, err := u.repo.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrPostNotFound
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		return err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		u.cache.InvalidatePost(ctx, id)
		u.cache.InvalidatePosts(ctx)

	}()

	return nil
}
