package postgres

import (
	"context"
	loggerInterface "github.com/col3name/lines/pkg/common/application/logger"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pashagolub/pgxmock"
	"testing"
)

func GetPgxMockPool(t *testing.T) (pgxmock.PgxPoolIface, error) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
	}
	return mock, err
}

func CheckExpectationsWereMet(t *testing.T, mock pgxmock.PgxPoolIface) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func SetupDbConnection(dbUrl string, logger loggerInterface.Logger) PgxPoolIface {
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		logger.Fatal("Unable to parse DATABASE_URL", "error", err)
	}
	db, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", "error", err)
	}
	return db
}
