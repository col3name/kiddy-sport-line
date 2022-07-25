package application

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
	log "github.com/sirupsen/logrus"
)

type SportLineService interface {
	Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.Sport, error)
	IsChanged(exist bool, oldValue map[commonDomain.SportType]float32, newValue []commonDomain.SportType) bool
}

type SportLineServiceImpl struct {
	sportRepo domain.SportRepo
}

func NewSportLineService(repo domain.SportRepo) *SportLineServiceImpl {
	return &SportLineServiceImpl{sportRepo: repo}
}

func (s *SportLineServiceImpl) IsChanged(exist bool, oldValue map[commonDomain.SportType]float32, sports []commonDomain.SportType) bool {
	isSubChanged := true
	if exist {
		if len(oldValue) != len(sports) {
			isSubChanged = true
		} else {
			for _, sportType := range sports {
				_, ok := oldValue[sportType]
				if !ok {
					isSubChanged = true
					break
				}
			}
		}
	}

	return isSubChanged
}
func (s *SportLineServiceImpl) Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*domain.Sport, error) {
	sportLines, err := s.sportRepo.GetSportLines(sports)
	if err != nil {
		log.Println(err)
		return []*domain.Sport{}, err
	}
	return s.calculateLineOfSports(sportLines, isNeedDelta, subs), nil
}
