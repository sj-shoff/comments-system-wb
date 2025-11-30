package posts

import "github.com/wb-go/wbf/zlog"

type PostsHandler struct {
	usecase postsUsecase
	logger  *zlog.Zerolog
}
