package repository

import (
	"context"
	"github.com/col3name/lines/pkg/common/application/logger"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/col3name/lines/pkg/common/infrastructure/postgres"
	"github.com/jackc/pgx/v4"
	"time"
)

type Database struct {
	conn postgres.PgxPoolIface
}

func NewDatabase(conn postgres.PgxPoolIface) *Database {
	return &Database{conn: conn}
}

func (db *Database) WithTx(job func(pgx.Tx) error, logger logger.Logger) (context.CancelFunc, error) {
	timeout, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	tx, err := db.conn.Begin(timeout)
	if err != nil {
		return cancel, infrastructure.InternalError(logger, err)
	}
	err = job(tx)
	if err != nil {
		err2 := tx.Rollback(timeout)
		if err2 != nil {
			logger.Error(err2)
		}
	} else {
		err2 := tx.Commit(timeout)
		if err2 != nil {
			logger.Error(err2)
			return cancel, err2
		}
	}

	return cancel, err
}
