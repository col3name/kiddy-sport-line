package application

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
)

type SportTypeMap map[commonDomain.SportType]float32

type SportLineService interface {
	Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*commonDomain.SportLine, error)
	IsChanged(exist bool, subscriptionMap SportTypeMap, newValue []commonDomain.SportType) bool
}

type sportLineServiceImpl struct {
	sportRepo domain.SportRepo
}

func NewSportLineService(repo domain.SportRepo) *sportLineServiceImpl {
	return &sportLineServiceImpl{sportRepo: repo}
}

func (s *sportLineServiceImpl) Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*commonDomain.SportLine, error) {
	if subs == nil {
		return nil, errors.ErrInvalidArgument
	}
	sportLines, err := s.sportRepo.GetLinesBySportTypes(sports)
	if err != nil {
		return nil, err
	}
	return s.calculateLineOfSports(sportLines, isNeedDelta, subs), nil
}

func (s *sportLineServiceImpl) IsChanged(exist bool, subMap SportTypeMap, sports []commonDomain.SportType) bool {
	if subMap == nil || len(sports) == 0 {
		return false
	}
	isSubChanged := true
	if exist {
		isSubChanged = !s.isEqual(subMap, sports)
	}

	return isSubChanged
}

func (s *sportLineServiceImpl) isEqual(oldValue SportTypeMap, sports []commonDomain.SportType) bool {
	if len(oldValue) != len(sports) {
		return false
	}
	for _, sportType := range sports {
		_, ok := oldValue[sportType]
		if !ok {
			return false
		}
	}
	return true
}
