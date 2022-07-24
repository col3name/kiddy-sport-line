package domain

import "github.com/col3name/lines/pkg/common/domain"

type SportRepo interface {
	GetSportLines(sportTypes []domain.SportType) ([]domain.SportLine, error)
	Store(model *domain.SportLine) error
}
