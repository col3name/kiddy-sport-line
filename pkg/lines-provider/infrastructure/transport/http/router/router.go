package router

import (
	"fmt"
	httpUtil "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"github.com/col3name/lines/pkg/lines-provider/application/service"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

func Router(scoreService service.ScoreService) *mux.Router {
	controller := sportLineController{scoreService: scoreService}

	router := mux.NewRouter()

	router.HandleFunc("/ready", httpUtil.ReadyCheckHandler).Methods(http.MethodGet)

	apiV1Route := router.PathPrefix("/api/v1").Subrouter()
	apiV1Route.HandleFunc("/lines/{sport}", controller.getSportLineHandler).Methods(http.MethodGet)

	return router
}

type sportLineController struct {
	scoreService service.ScoreService
}

func (c *sportLineController) getSportLineHandler(w http.ResponseWriter, req *http.Request) {
	sport := c.parseRequest(req)

	score, err := c.scoreService.GenerateScore(strings.ToLower(sport))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	httpUtil.WriteJSON(w, c.marshalScore(sport, score))
}

func (c *sportLineController) parseRequest(req *http.Request) string {
	vars := mux.Vars(req)
	return vars["sport"]
}

func (c *sportLineController) marshalScore(sport string, score float64) string {
	return fmt.Sprintf("{\"lines\": {\"%s\":\"%f\"}}", strings.ToUpper(sport), score)
}
