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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/comments", func(r chi.Router) {
		r.Post("/", h.CommentsHandler.CreateComment)
		r.Get("/", h.CommentsHandler.GetComments)
		r.Delete("/{id}", h.CommentsHandler.DeleteComment)
	})

	r.Route("/posts", func(r chi.Router) {
		r.Post("/", h.PostsHandler.CreatePost)
		r.Get("/", h.PostsHandler.GetPosts)
		r.Get("/{id}", h.PostsHandler.GetPostByID)
		r.Delete("/{id}", h.PostsHandler.DeletePost)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
