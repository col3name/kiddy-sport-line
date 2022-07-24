package main

import (
	"fmt"
	"github.com/col3name/lines/pkg/common/domain"
	netHttp "github.com/col3name/lines/pkg/common/infrastructure/transport/net-http"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	var err error
	getenv := os.Getenv("PORT")
	port := 8000
	if len(getenv) > 0 {
		port, err = strconv.Atoi(getenv)
		if err != nil {
			log.Fatalf("Invalid port %s", getenv)
		}
	}
	r := mux.NewRouter()
	apiV1Route := r.PathPrefix("/api/v1").Subrouter()
	r.HandleFunc("/ready", netHttp.ReadyCheckHandler).Methods(http.MethodGet)
	apiV1Route.HandleFunc("/lines/{sport}", getLineSport)
	serverUrl := ":" + strconv.Itoa(port)
	log.Info("listen and serve at", serverUrl)
	_ = http.ListenAndServe(serverUrl, logMiddleware(r))
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

func logMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"method":     r.Method,
			"url":        r.URL,
			"remoteAddr": r.RemoteAddr,
			"userAgent":  r.UserAgent(),
		}).Info("got a new request")
		h.ServeHTTP(w, r)
	})
}
