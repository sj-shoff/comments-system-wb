package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"comments-system/internal/domain"
	"comments-system/internal/usecase/comments"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
)

type CommentsRepository struct {
	db      *dbpg.DB
	retries retry.Strategy
}

func NewCommentsRepository(db *dbpg.DB, retries retry.Strategy) *CommentsRepository {
	return &CommentsRepository{
		db:      db,
		retries: retries,
	}
}

func (r *CommentsRepository) Create(ctx context.Context, comment domain.Comment) (domain.Comment, error) {
	var id int
	var createdAt, updatedAt time.Time
	query := `INSERT INTO comments (parent_id, content, author, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id, created_at, updated_at`
	row, err := r.db.QueryRowWithRetry(ctx, r.retries, query, comment.ParentID, comment.Content, comment.Author)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("failed to query row: %w", err)
	}
	err = row.Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("failed to scan: %w", err)
	}
	comment.ID = id
	comment.CreatedAt = createdAt
	comment.UpdatedAt = updatedAt
	return comment, nil
}

func (r *CommentsRepository) Exists(ctx context.Context, id int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE id = $1`
	row, err := r.db.QueryRowWithRetry(ctx, r.retries, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to query row: %w", err)
	}
	err = row.Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to scan: %w", err)
	}
	return count > 0, nil
}

func (r *CommentsRepository) Delete(ctx context.Context, id int) error {
	query := `
	WITH RECURSIVE descendants AS (
	  SELECT id FROM comments WHERE id = $1
	  UNION
	  SELECT c.id FROM comments c INNER JOIN descendants d ON c.parent_id = d.id
	)
	DELETE FROM comments WHERE id IN (SELECT id FROM descendants)
	`
	_, err := r.db.ExecWithRetry(ctx, r.retries, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	return nil
}

func (r *CommentsRepository) GetTree(ctx context.Context, rootID int) ([]domain.Comment, error) {
	query := `
	WITH RECURSIVE tree AS (
	  SELECT id, parent_id, content, author, created_at, updated_at FROM comments WHERE id = $1
	  UNION
	  SELECT c.id, c.parent_id, c.content, c.author, c.created_at, c.updated_at FROM comments c INNER JOIN tree t ON c.parent_id = t.id
	)
	SELECT id, parent_id, content, author, created_at, updated_at FROM tree ORDER BY id
	`
	rows, err := r.db.QueryWithRetry(ctx, r.retries, query, rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comment tree: %w", err)
	}
	defer rows.Close()

	commentMap := make(map[int]*domain.Comment)
	for rows.Next() {
		var c domain.Comment
		var parentID sql.NullInt32
		err := rows.Scan(&c.ID, &parentID, &c.Content, &c.Author, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment row: %w", err)
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			c.ParentID = &pid
		}
		commentMap[c.ID] = &c
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comment rows: %w", err)
	}

	root, ok := commentMap[rootID]
	if !ok {
		return nil, comments.ErrCommentNotFound
	}

	for _, c := range commentMap {
		if c.ParentID != nil {
			parent, ok := commentMap[*c.ParentID]
			if ok {
				parent.Children = append(parent.Children, *c)
			}
		}
	}

	return root.Children, nil
}

func (r *CommentsRepository) GetByID(ctx context.Context, id int) (domain.Comment, error) {
	var c domain.Comment
	var parentID sql.NullInt32
	query := `SELECT id, parent_id, content, author, created_at, updated_at FROM comments WHERE id = $1`
	row, err := r.db.QueryRowWithRetry(ctx, r.retries, query, id)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("failed to query row: %w", err)
	}
	err = row.Scan(&c.ID, &parentID, &c.Content, &c.Author, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Comment{}, comments.ErrCommentNotFound
	}
	if err != nil {
		return domain.Comment{}, fmt.Errorf("failed to scan: %w", err)
	}
	if parentID.Valid {
		pid := int(parentID.Int32)
		c.ParentID = &pid
	}
	return c, nil
}

func (r *CommentsRepository) GetByParent(ctx context.Context, parentID *int, page, pageSize int, searchQuery, sortBy, sortOrder string) ([]domain.Comment, int, error) {
	where := "WHERE parent_id IS NULL"
	params := []interface{}{}
	i := 1
	if searchQuery != "" {
		where += fmt.Sprintf(" AND content ILIKE $%d", i)
		params = append(params, "%"+searchQuery+"%")
		i++
	}

	countQuery := `SELECT COUNT(*) FROM comments ` + where
	row, err := r.db.QueryRowWithRetry(ctx, r.retries, countQuery, params...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query row: %w", err)
	}
	var total int
	err = row.Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan: %w", err)
	}

	sortField := "created_at"
	switch sortBy {
	case "id":
		sortField = "id"
	case "updated_at":
		sortField = "updated_at"
	}
	sortDir := "DESC"
	if sortOrder == "asc" {
		sortDir = "ASC"
	}

	query := `SELECT id, parent_id, content, author, created_at, updated_at FROM comments ` + where + ` ORDER BY ` + sortField + ` ` + sortDir + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, i, i+1)
	params = append(params, pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryWithRetry(ctx, r.retries, query, params...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		var pid sql.NullInt32
		err := rows.Scan(&c.ID, &pid, &c.Content, &c.Author, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan comment row: %w", err)
		}
		if pid.Valid {
			pidInt := int(pid.Int32)
			c.ParentID = &pidInt
		}
		comments = append(comments, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, total, nil
}
