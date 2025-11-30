package comments

import "github.com/wb-go/wbf/zlog"

type CommentsHandler struct {
	usecase commentsUsecase
	logger  *zlog.Zerolog
}
