package sport_line

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/adapter"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
)

type SportLinesUpdateService interface {
	Update(sportType commonDomain.SportType) error
}

type sportLinesUpdateService struct {
	updatePeriod         int
	linesProviderAdapter adapter.LinesProviderAdapter
	uow                  service.UnitOfWork
}

func NewSportLinesUpdateService(updatePeriod int, linesProviderAdapter adapter.LinesProviderAdapter, uow service.UnitOfWork) *sportLinesUpdateService {
	return &sportLinesUpdateService{
		updatePeriod:         updatePeriod,
		linesProviderAdapter: linesProviderAdapter,
		uow:                  uow,
	}
}

func (s *sportLinesUpdateService) Update(sportType commonDomain.SportType) error {
	sportLine, err := s.linesProviderAdapter.GetLineBySport(sportType)
	if err != nil {
		return err
	}

	job := func(rp service.RepositoryProvider) error {
		sportLineRepo := rp.SportLineRepo()
		return sportLineRepo.Store(sportLine)
	}

	return s.uow.Execute(job)
}
