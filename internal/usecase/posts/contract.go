package posts

import (
	"context"

	"comments-system/internal/domain"
)

type postsRepo interface {
	Create(ctx context.Context, post domain.Post) (domain.Post, error)
	GetAll(ctx context.Context, page, pageSize int, searchQuery, sortBy, sortOrder string) ([]domain.Post, int, error)
	GetByID(ctx context.Context, id int) (domain.Post, error)
	Delete(ctx context.Context, id int) error
	Exists(ctx context.Context, id int) (bool, error)
	GetCommentsCount(ctx context.Context, postID int) (int, error)
}

type postsCache interface {
	GetPost(ctx context.Context, id int) (domain.Post, error)
	SetPost(ctx context.Context, post domain.Post) error
	InvalidatePost(ctx context.Context, id int) error
	GetPosts(ctx context.Context, page, pageSize int, searchQuery, sortBy, sortOrder string) ([]domain.Post, int, error)
	SetPosts(ctx context.Context, page, pageSize int, searchQuery, sortBy, sortOrder string, posts []domain.Post, totalCount int) error
	InvalidatePosts(ctx context.Context) error
	GetCommentsCount(ctx context.Context, postID int) (int, error)
	SetCommentsCount(ctx context.Context, postID int, count int) error
}
