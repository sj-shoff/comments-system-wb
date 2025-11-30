package domain

import "time"

type Post struct {
	ID            int
	Title         string
	Content       string
	Author        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CommentsCount int
}

type PostTree struct {
	Posts    []Post
	Total    int
	Page     int
	PageSize int
	HasNext  bool
	HasPrev  bool
}
