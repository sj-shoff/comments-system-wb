package comments

import (
	"context"

	"comments-system/internal/domain"
)

type сommentsRepo interface {
	Create(ctx context.Context, comment domain.Comment) (domain.Comment, error)
	GetByParent(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string) ([]domain.Comment, int, error)
	GetTree(ctx context.Context, rootID int) ([]domain.Comment, error)
	Delete(ctx context.Context, id int) error
	Exists(ctx context.Context, id int) (bool, error)
	GetByID(ctx context.Context, id int) (domain.Comment, error)
}

type commentsCache interface {
	GetCommentTree(ctx context.Context, rootID int) ([]domain.Comment, error)
	SetCommentTree(ctx context.Context, rootID int, comments []domain.Comment) error
	InvalidateCommentTree(ctx context.Context, rootID int) error
	GetComments(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string) ([]domain.Comment, int, error)
	SetComments(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string, comments []domain.Comment, totalCount int) error
	InvalidateComments(ctx context.Context, parentID *int) error
}
