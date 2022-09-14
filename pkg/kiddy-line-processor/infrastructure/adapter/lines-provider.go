package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/col3name/lines/pkg/common/application/logger"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure"
	http2 "github.com/col3name/lines/pkg/common/infrastructure/transport/http"
	"io"
	"net/http"
)

type BaseSport struct {
}

type BaseballResp struct {
	BaseSport
	Lines struct {
		Score string `json:"BASEBALL"`
	} `json:"lines"`
}

type FootballResp struct {
	BaseSport
	Lines struct {
		Score string `json:"FOOTBALL"`
	} `json:"lines"`
}

type SoccerResp struct {
	BaseSport
	Lines struct {
		Score string `json:"SOCCER"`
	} `json:"lines"`
}

type linesProviderAdapter struct {
	linesProviderUrl string
	logger           logger.Logger
}

func NewLinesProviderAdapter(linesProviderUrl string, logger logger.Logger) *linesProviderAdapter {
	return &linesProviderAdapter{linesProviderUrl: linesProviderUrl, logger: logger}
}

func (s linesProviderAdapter) GetLineBySport(sportType commonDomain.SportType) (*commonDomain.SportLine, error) {
	url := s.getLinesURL(sportType)
	resp, err := http2.Get(url)
	if err != nil {
		return nil, infrastructure.ExternalError(s.logger, err)
	}
	return s.parseResp(resp, sportType)
}

func (s linesProviderAdapter) getLinesURL(sportType commonDomain.SportType) string {
	return fmt.Sprintf("%s/api/v1/lines/%s", s.linesProviderUrl, sportType)
}

func (s *linesProviderAdapter) parseResp(resp *http.Response, sportType commonDomain.SportType) (*commonDomain.SportLine, error) {
	if resp.StatusCode != http.StatusOK {
		err := s.failedGetSportError(sportType, nil)
		return nil, infrastructure.ExternalError(s.logger, err)
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		err = s.failedGetSportError(sportType, err)
		return nil, infrastructure.InternalError(s.logger, err)
	}
	defer resp.Body.Close()

	sportLine, err := s.parseGetLinesResponse(bytes, sportType)
	if err != nil {
		return nil, infrastructure.InternalError(s.logger, err)
	}
	return sportLine, nil
}

func (s *linesProviderAdapter) failedGetSportError(sportType commonDomain.SportType, err error) error {
	text := "failed get " + string(sportType) + "data"
	if err != nil {
		text += err.Error()
	}
	return errors.New(text)
}

func (s *linesProviderAdapter) parseGetLinesResponse(bytes []byte, sportType commonDomain.SportType) (*commonDomain.SportLine, error) {
	var (
		sport commonDomain.SportLine
		score string
		err   error
	)
	switch sportType {
	case commonDomain.Baseball:
		var model BaseballResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Baseball
		score = model.Lines.Score
	case commonDomain.Soccer:
		var model SoccerResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Soccer
		score = model.Lines.Score
	case commonDomain.Football:
		var model FootballResp
		err = json.Unmarshal(bytes, &model)
		if err != nil {
			return nil, err
		}
		sport.Type = commonDomain.Football
		score = model.Lines.Score
	}
	err = sport.SetScore(score)

	return &sport, err
}
