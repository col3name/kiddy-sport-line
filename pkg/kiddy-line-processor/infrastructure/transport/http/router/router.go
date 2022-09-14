package router

import (
	"github.com/col3name/lines/pkg/common/application/logger"
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/gorilla/mux"
	"net/http"
)

func Router(logger logger.Logger) http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/ready", httpUtil.ReadyCheckHandler)

	return httpUtil.LogMiddleware(router, logger)
}
