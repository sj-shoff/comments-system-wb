package router

import (
	"net/http"

	"comments-system/internal/http-server/handler/comments"
	"comments-system/internal/http-server/handler/posts"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	CommentsHandler *comments.CommentsHandler
	PostsHandler    *posts.PostsHandler
}

func SetupRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	return r
}
