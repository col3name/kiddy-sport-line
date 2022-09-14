package repo

import (
	"context"
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	postgres2 "github.com/col3name/lines/pkg/kiddy-line-processor/infrastructure/postgres"
	"github.com/jackc/pgx/v4"
)

type SportLineRepoImpl struct {
	conn   postgres.PgxPoolIface
	logger logger.Logger
}

func NewSportLineRepository(conn postgres.PgxPoolIface, logger logger.Logger) *SportLineRepoImpl {
	return &SportLineRepoImpl{conn: conn, logger: logger}
}

func (r *SportLineRepoImpl) Store(model *domain.SportLine) error {
	const sql = "UPDATE sport_lines SET score = $1 WHERE sport_type = $2;"

	job := func(tx pgx.Tx) error {
		result, err := tx.Exec(context.Background(), sql, model.Score, model.Type)
		if err != nil {
			return err
		}
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return domain.ErrSportLinesDoesNotExist
		}
		return nil
	}
	cancelFunc, err := postgres2.WithTx(r.conn, job, r.logger)
	if cancelFunc != nil {
		defer cancelFunc()
	}
	if err != nil {
		return infrastructure.InternalError(r.logger, err)
	}
	return nil
}
