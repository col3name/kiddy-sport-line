package router

import (
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/gorilla/mux"
	"net/http"
)

func Router() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/ready", httpUtil.ReadyCheckHandler)
	return httpUtil.LogMiddleware(router, s.logger)
}
