package main

import (
	"fmt"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/env"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/col3name/lines/pkg/common/infrastructure/util/number"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

func main() {
	logService := logrusLogger.New()

	port := env.GetEnvVariable("PORT", "8000")

	serverUrl := ":" + port
	logService.Info("listen and serve at", serverUrl)
	defer logService.Info("stop")

	router := buildRouter()
	handler := httpUtil.LogMiddleware(router, logService)

	httpUtil.RunHttpServer(serverUrl, handler, logService)
}

func buildRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/ready", httpUtil.ReadyCheckHandler).Methods(http.MethodGet)

	apiV1Route := router.PathPrefix("/api/v1").Subrouter()
	apiV1Route.HandleFunc("/lines/{sport}", getLineSport).Methods(http.MethodGet)

	return router
}

func getLineSport(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sport := vars["sport"]
	lower := strings.ToLower(sport)
	_, exist := domain.SupportSports[lower]
	if !exist {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	upper := strings.ToUpper(sport)
	_, _ = w.Write([]byte(fmt.Sprintf("{\"lines\": {\"%s\":\"%f\"}}", upper, number.RandFloat(0.5, 3))))
}
