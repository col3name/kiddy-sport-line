package domain

import "github.com/col3name/lines/pkg/common/domain"

type SportRepo interface {
	Store(model *domain.SportLine) error
	GetSportLines(sportTypes []domain.SportType) ([]domain.SportLine, error)
}
