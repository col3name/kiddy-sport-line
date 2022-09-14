package query

import (
	"context"
	"fmt"
	appErr "github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/query"
	"github.com/jackc/pgx/v4"
	"strings"
)

type SportLineQueryServiceImpl struct {
	conn   postgres.PgxPoolIface
	logger logger.Logger
}

func NewSportLineQueryService(conn postgres.PgxPoolIface, logger logger.Logger) query.SportLineQueryService {
	return &SportLineQueryServiceImpl{conn: conn, logger: logger}
}

func (r *SportLineQueryServiceImpl) GetLinesBySportTypes(sportTypes []domain.SportType) ([]*domain.SportLine, error) {
	countSportTypes := len(sportTypes)
	if countSportTypes < 1 {
		return nil, appErr.ErrInvalidArgument
	}
	sql, data := r.getSqlQueryAndData(sportTypes, countSportTypes)
	rows, err := r.conn.Query(context.Background(), sql, data...)
	if err != nil {
		if r.isTableNotExistError(err) {
			return nil, appErr.ErrTableNotExist
		}
		return nil, infrastructure.InternalError(r.logger, err)
	}
	if rows.Err() != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSportLines(rows)
}

func (r *SportLineQueryServiceImpl) getSqlQueryAndData(sportTypes []domain.SportType, countSportTypes int) (string, []interface{}) {
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

func (r *SportLineQueryServiceImpl) getSqlSelectSportType(i int) string {
	return fmt.Sprintf("SELECT score,sport_type FROM sport_lines WHERE sport_type = $%d ", i)
}

func (r *SportLineQueryServiceImpl) isTableNotExistError(err error) bool {
	return strings.Contains(err.Error(), appErr.TableNotExistMessage)
}

func (r *SportLineQueryServiceImpl) scanSportLines(rows pgx.Rows) ([]*domain.SportLine, error) {
	var sport domain.SportLine
	var sports []*domain.SportLine
	var err error
	for rows.Next() {
		err = rows.Scan(&sport.Score, &sport.Type)
		if err != nil {
			return sports, infrastructure.InternalError(r.logger, err)
		}
		sports = append(sports, &sport)
	}
	return sports, nil
}
