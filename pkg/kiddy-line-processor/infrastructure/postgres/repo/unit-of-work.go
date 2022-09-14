package repo

import (
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/col3name/lines/pkg/common/infrastructure/repository"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/service"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/repo"
	"github.com/jackc/pgx/v4"
)

type unitOfWork struct {
	db     *repository.Database
	logger logger.Logger
}

func NewUnitOfWork(conn postgres.PgxPoolIface, logger logger.Logger) *unitOfWork {
	return &unitOfWork{
		db:     repository.NewDatabase(conn),
		logger: logger,
	}
}

func (u *unitOfWork) Execute(fn service.Job) error {
	cancelFunc, err := u.db.WithTx(func(tx pgx.Tx) error {
		return fn(&repositoryProvider{tx: tx})
	}, u.logger)
	if err != nil {
		return infrastructure.InternalError(u.logger, err)
	}
	defer cancelFunc()
	return nil
}

type repositoryProvider struct {
	tx     pgx.Tx
	logger logger.Logger
}

func (r *repositoryProvider) MigrationRepo() repo.MigrationRepo {
	return NewMigrationRepo(r.tx)
}

func (r *repositoryProvider) SportLineRepo() repo.SportLineRepo {
	return NewSportLineRepository(r.tx, r.logger)
}
