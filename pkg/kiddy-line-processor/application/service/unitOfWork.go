package service

import (
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain/repo"
)

type Job func(RepositoryProvider) error

type RepositoryProvider interface {
	SportLineRepo() repo.SportLineRepo
	MigrationRepo() repo.MigrationRepo
}

type UnitOfWork interface {
	Execute(fn Job) error
}
