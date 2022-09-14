package repo

import (
	"context"
	"github.com/jackc/pgx/v4"
)

const CreateSportLinesSql = `BEGIN TRANSACTION;
				CREATE TABLE sport_lines
				(
					id         UUID PRIMARY KEY UNIQUE NOT NULL,
					sport_type VARCHAR(255)            NOT NULL,
					score      REAL                     NOT NULL
				);
				
				INSERT INTO sport_lines (id, sport_type, score)
				VALUES ('ce267749-dec9-4d39-ad81-8b4cd8c381d2', 'baseball', 1.0),
					   ('ba9babe8-06d4-450e-8e9a-66b7512b5bd2', 'soccer', 1.0),
					   ('4b9d52e2-1473-4cdb-bba8-c1c1cac933f5', 'football', 1.0);
				END ;`

type migration struct {
	tx pgx.Tx
}

func NewMigrationRepo(tx pgx.Tx) *migration {
	return &migration{tx: tx}
}

func (m *migration) Migrate() error {
	_, err := m.tx.Exec(context.Background(), CreateSportLinesSql)
	return err
}
