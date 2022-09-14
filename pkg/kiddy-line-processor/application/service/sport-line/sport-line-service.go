package sport_line

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure/util/array"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/model"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/query"
)

type SportLineService interface {
	Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*commonDomain.SportLine, error)
	IsSubscriptionChanged(exist bool, subscriptionMap model.SportTypeMap, newValue []commonDomain.SportType) bool
}

type sportLineServiceImpl struct {
	sportLineQueryService query.SportLineQueryService
}

func NewSportLineService(queryService query.SportLineQueryService) *sportLineServiceImpl {
	return &sportLineServiceImpl{sportLineQueryService: queryService}
}

func (s *sportLineServiceImpl) Calculate(sports []commonDomain.SportType, isNeedDelta bool, subs *model.ClientSubscription) ([]*commonDomain.SportLine, error) {
	if subs == nil {
		return nil, errors.ErrInvalidArgument
	}
	sportLines, err := s.sportLineQueryService.GetLinesBySportTypes(sports)
	if err != nil {
		return nil, err
	}
	return s.calculateLineOfSports(sportLines, isNeedDelta, subs), nil
}

func (s *sportLineServiceImpl) calculateLineOfSports(lines []*commonDomain.SportLine, isNeedDelta bool, subs *model.ClientSubscription) []*commonDomain.SportLine {
	for i, line := range lines {
		s.calculateLine(line, isNeedDelta, subs)
		lines[i] = line
	}

	return lines
}

func (s *sportLineServiceImpl) calculateLine(line *commonDomain.SportLine, isNeedDelta bool, subs *model.ClientSubscription) {
	sportType := line.Type
	if isNeedDelta {
		line.Score = line.Score - subs.Sports[sportType]
	}
	subs.Sports[sportType] = line.Score
}

func (s *sportLineServiceImpl) IsSubscriptionChanged(exist bool, subMap model.SportTypeMap, sports []commonDomain.SportType) bool {
	return !s.isValidSubscription(subMap, sports) && s.isSubscriptionEqual(exist, subMap, sports)
}

func (s *sportLineServiceImpl) isSubscriptionEqual(exist bool, oldSubscription model.SportTypeMap, newSubscription []commonDomain.SportType) bool {
	return !exist || exist && !s.compareOldAndNewSubscription(oldSubscription, newSubscription)
}

func (s *sportLineServiceImpl) isValidSubscription(subMap model.SportTypeMap, sports []commonDomain.SportType) bool {
	return subMap == nil || array.EmptyST(sports)
}

func (s *sportLineServiceImpl) compareOldAndNewSubscription(oldSubscription model.SportTypeMap, newSubscription []commonDomain.SportType) bool {
	if len(oldSubscription) != len(newSubscription) {
		return false
	}
	for _, sportType := range newSubscription {
		if _, ok := oldSubscription[sportType]; !ok {
			return false
		}
	}
	return true
}
