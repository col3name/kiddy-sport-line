package pg

import (
	"github.com/col3name/lines/pkg/common/application/errors"
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
	domainQuery "github.com/col3name/lines/pkg/kiddy-line-processor/domain/query"
)

type MigrationService interface {
	MigrateIfNeeded() error
}

type migrationService struct {
	sportLineQueryService domainQuery.SportLineQueryService
	uow                   service.UnitOfWork
}

func NewMigrationService(sportLineQueryService domainQuery.SportLineQueryService, uow service.UnitOfWork) MigrationService {
	return &migrationService{
		sportLineQueryService: sportLineQueryService,
		uow:                   uow,
	}
}

func (s *migrationService) MigrateIfNeeded() error {
	defaultSubscriptions := []commonDomain.SportType{commonDomain.Baseball}
	_, err := s.sportLineQueryService.GetLinesBySportTypes(defaultSubscriptions)
	if err == nil {
		return nil
	}
	if err != errors.ErrTableNotExist {
		return err
	}

	return s.uow.Execute(func(provider service.RepositoryProvider) error {
		migrationRepo := provider.MigrationRepo()
		return migrationRepo.Migrate()
	})
}
