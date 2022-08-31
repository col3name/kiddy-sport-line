package postgres

import (
	"context"
	"fmt"
	appErr "github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/jackc/pgx/v4"
	"strings"
)

type SportRepoImpl struct {
	conn   postgres.PgxPoolIface
	logger logger.Logger
}

func NewSportLineRepository(conn postgres.PgxPoolIface, logger logger.Logger) *SportRepoImpl {
	return &SportRepoImpl{conn: conn, logger: logger}
}

func (r *SportRepoImpl) GetLinesBySportTypes(sportTypes []domain.SportType) ([]*domain.SportLine, error) {
	countSportTypes := len(sportTypes)
	if countSportTypes < 1 {
		return nil, appErr.ErrInvalidArgument
	}
	sql, data := r.getSqlQueryAndData(sportTypes, countSportTypes)
	rows, err := r.conn.Query(context.Background(), sql, data...)
	if err != nil {
		contains := strings.Contains(err.Error(), appErr.TableNotExistMessage)
		if contains {
			return nil, appErr.ErrTableNotExist
		}
		return nil, infrastructure.InternalError(r.logger, err)
	}
	if rows.Err() != nil {
		return nil, err
	}
	defer rows.Close()

	var sport domain.SportLine
	var sports []*domain.SportLine
	for rows.Next() {
		err = rows.Scan(&sport.Score, &sport.Type)
		if err != nil {
			return sports, infrastructure.InternalError(r.logger, err)
		}
		sports = append(sports, &sport)
	}
	return sports, nil
}

func (r *SportRepoImpl) Store(model *domain.SportLine) error {
	const sql = "UPDATE sport_lines SET score = $1 WHERE sport_type = $2;"

	job := func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), sql, model.Score, model.Type)
		return err
	}
	cancelFunc, err := WithTx(r.conn, job, r.logger)
	if cancelFunc != nil {
		defer cancelFunc()
	}
	if err != nil {
		return infrastructure.InternalError(r.logger, err)
	}
	return nil
}

func (r *SportRepoImpl) getSqlQueryAndData(sportTypes []domain.SportType, countSportTypes int) (string, []interface{}) {
	var sql string
	var data []interface{}
	sql = r.getSqlSelectSportType(1)
	data = append(data, sportTypes[0])
	if countSportTypes > 1 {
		for i := 1; i < countSportTypes; i++ {
			sql += ` UNION ALL `
			sql += r.getSqlSelectSportType(i + 1)
			data = append(data, sportTypes[i])
		}
	}
	sql += ";"
	return sql, data
}

func (r *SportRepoImpl) getSqlSelectSportType(i int) string {
	return fmt.Sprintf("SELECT score,sport_type FROM sport_lines WHERE sport_type = $%d ", i)
}
