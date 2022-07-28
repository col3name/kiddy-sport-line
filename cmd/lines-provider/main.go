package main

import (
	"fmt"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"

	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/logrusLogger"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	logger := logrusLogger.New()

	var err error
	portStr := os.Getenv("PORT")
	port := 8000
	if len(portStr) > 0 {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			logger.Fatalf("Invalid port %s", portStr)
		}
	}
	r := mux.NewRouter()
	apiV1Route := r.PathPrefix("/api/v1").Subrouter()
	r.HandleFunc("/ready", netHttp.ReadyCheckHandler).Methods(http.MethodGet)
	apiV1Route.HandleFunc("/lines/{sport}", getLineSport)
	serverUrl := ":" + strconv.Itoa(port)
	logger.Info("listen and serve at", serverUrl)
	_ = http.ListenAndServe(serverUrl, logMiddleware(r, logger))
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
	_, _ = w.Write([]byte(fmt.Sprintf("{\"lines\": {\"%s\":\"%f\"}}", upper, randFloat(0.5, 3))))
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func logMiddleware(h http.Handler, logger loggerInterface.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.With(loggerInterface.Fields{
			"method":     r.Method,
			"url":        r.URL,
			"remoteAddr": r.RemoteAddr,
			"userAgent":  r.UserAgent(),
		}).Info("got a new request")
		h.ServeHTTP(w, r)
	})
}
