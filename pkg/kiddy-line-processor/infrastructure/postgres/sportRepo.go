package postgres

import (
	"context"
	"errors"
	"fmt"
	appErr "github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"strings"
)

const TableNotExistMessage = " does not exist (SQLSTATE 42P01)"

var (
	ErrTableNotExist = errors.New("table does not exist")
)

type SportRepoImpl struct {
	conn *pgxpool.Pool
}

func NewSportLineRepository(conn *pgxpool.Pool) *SportRepoImpl {
	u := new(SportRepoImpl)
	u.conn = conn
	return u
}

//sql := `SELECT score,sport_type FROM sport_lines WHERE sport_type = 'baseball'
//UNION ALL
//SELECT score, sport_type FROM sport_lines WHERE sport_type = 'football'`
func (r *SportRepoImpl) GetSportLines(sportTypes []domain.SportType) ([]domain.SportLine, error) {
	countSportTypes := len(sportTypes)
	if countSportTypes < 1 {
		return nil, appErr.ErrInvalidArgument
	}
	var sql string
	getSqlSelectSportType := func(i int) string {
		return fmt.Sprintf("SELECT score,sport_type FROM sport_lines WHERE sport_type = $%d ", i)
	}
	var data []interface{}
	sql = getSqlSelectSportType(1)
	data = append(data, sportTypes[0])
	if countSportTypes > 1 {
		for i := 1; i < countSportTypes; i++ {
			sql += `UNION ALL `
			sql += getSqlSelectSportType(i + 1)
			data = append(data, sportTypes[i])
		}
	}
	sql += ";"

	rows, err := r.conn.Query(context.Background(), sql, data...)
	if err != nil {
		s := err.Error()
		contains := strings.Contains(s, TableNotExistMessage)
		if contains {
			return nil, ErrTableNotExist
		}
		return nil, infrastructure.InternalError(err)
	}
	if rows.Err() != nil {
		return nil, err
	}
	defer rows.Close()

	var sport domain.SportLine
	var sports []domain.SportLine
	for rows.Next() {
		err = rows.Scan(&sport.Score, &sport.Type)
		if err != nil {
			return sports, infrastructure.InternalError(err)
		}
		sports = append(sports, sport)
	}
	return sports, nil
}

// CREATE TABLE sport_lines (
//   id         UUID PRIMARY KEY UNIQUE NOT NULL,
//   sport_type VARCHAR(255)     NOT NULL,
//   score      REAL              NOT NULL
// );
func (r *SportRepoImpl) Store(model *domain.SportLine) error {
	sql := "UPDATE sport_lines SET score = $1 WHERE sport_type = $2;"

	cancelFunc, err := WithTx(r.conn, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), sql, model.Score, model.Type)
		return err
	})
	if cancelFunc != nil {
		defer cancelFunc()
	}
	if err != nil {
		return infrastructure.InternalError(err)
	}
	return nil
}
