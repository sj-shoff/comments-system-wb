package postgres

import (
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
)

type PostsRepository struct {
	db      *dbpg.DB
	retries retry.Strategy
}

func NewPostsRepository(db *dbpg.DB, retries retry.Strategy) *PostsRepository {
	return &PostsRepository{
		db:      db,
		retries: retries,
	}
}
