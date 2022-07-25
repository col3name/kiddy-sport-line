package postgres

import (
	"context"
	"github.com/col3name/lines/pkg/common/infrastructure"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"time"
)

func WithTx(conn *pgxpool.Pool, job func(pgx.Tx) error) (context.CancelFunc, error) {
	timeout, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	tx, err := conn.Begin(timeout)
	if err != nil {
		return cancel, infrastructure.InternalError(err)
	}
	err = job(tx)
	if err != nil {
		err2 := tx.Commit(timeout)
		if err2 != nil {
			log.Error(err2)
		}
	} else {
		err2 := tx.Rollback(timeout)
		if err2 != nil {
			log.Error(err2)
		}
	}

	return cancel, err
}
