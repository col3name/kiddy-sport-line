package repo

type MigrationRepo interface {
	Migrate() error
}
