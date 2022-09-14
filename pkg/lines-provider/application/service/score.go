package service

import (
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/util/number"
)

type ScoreService interface {
	GenerateScore(sportType string) (float64, error)
}

type scoreService struct{}

func NewScoreService() ScoreService {
	return &scoreService{}
}

func (s *scoreService) GenerateScore(sportType string) (float64, error) {
	_, isSupported := domain.SupportSports[sportType]
	if !isSupported {
		return 0, domain.ErrUnsupportedSportType
	}
	score := number.RandFloat(0.5, 3)

	return score, nil
}
