package http

import (
	"context"
	"fmt"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func ReadyCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "{\"host\": \"%v\"}", r.Host)
}

func LogMiddleware(h http.Handler, logger loggerInterface.Logger) http.Handler {
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

func RunHttpServer(serverUrl string, handler http.Handler, logger loggerInterface.Logger) {
	srv := &http.Server{Addr: serverUrl, Handler: handler}
	killSignalChan := getKillSignalChan()

	err := srv.ListenAndServe()
	if err != nil {
		logger.Fatal(err)
	}
	<-killSignalChan
	err = srv.Shutdown(context.Background())
	if err != nil {
		logger.Fatal(err)
		return
	}
}

func getKillSignalChan() chan os.Signal {
	osKillSignalChan := make(chan os.Signal, 1)
	signal.Notify(osKillSignalChan, os.Interrupt, syscall.SIGTERM)

	return osKillSignalChan
}
