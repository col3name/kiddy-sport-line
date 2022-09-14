package repo

import (
	"context"
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/repo"
	"github.com/jackc/pgx/v4"
)

type sportLineRepo struct {
	tx     pgx.Tx
	logger logger.Logger
}

func NewSportLineRepository(tx pgx.Tx, logger logger.Logger) repo.SportLineRepo {
	return &sportLineRepo{tx: tx, logger: logger}
}

func (r *sportLineRepo) Store(model *domain.SportLine) error {
	const query = "UPDATE sport_lines SET score = $1 WHERE sport_type = $2;"

	result, err := r.tx.Exec(context.Background(), query, model.Score, model.Type)
	if err != nil {
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrSportLinesDoesNotExist
	}
	return nil
}
