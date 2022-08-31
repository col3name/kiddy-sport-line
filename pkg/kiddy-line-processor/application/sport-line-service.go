package application

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/util/array"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
)

type SportTypeMap map[commonDomain.SportType]float32

type SportLineService interface {
	Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *ClientSubscription) ([]*commonDomain.SportLine, error)
	IsSubscriptionChanged(exist bool, subscriptionMap SportTypeMap, newValue []commonDomain.SportType) bool
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

func (s *sportLineServiceImpl) IsSubscriptionChanged(exist bool, subMap SportTypeMap, sports []commonDomain.SportType) bool {
	return !s.isValidSubscription(subMap, sports) && s.isSubscriptionEqual(exist, subMap, sports)
}

func (s *sportLineServiceImpl) isSubscriptionEqual(exist bool, subMap SportTypeMap, sports []commonDomain.SportType) bool {
	return !exist || exist && !s.isSubsripitonEqual(subMap, sports)
}

func (s *sportLineServiceImpl) isValidSubscription(subMap SportTypeMap, sports []commonDomain.SportType) bool {
	return subMap == nil || array.EmptyST(sports)
}

func (s *sportLineServiceImpl) isSubsripitonEqual(oldSubscription SportTypeMap, newSubsription []commonDomain.SportType) bool {
	if len(oldSubscription) != len(newSubsription) {
		return false
	}
	for _, sportType := range newSubsription {
		if _, ok := oldSubscription[sportType]; !ok {
			return false
		}
	}
	return true
}
