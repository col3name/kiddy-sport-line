package query

import "github.com/col3name/lines/pkg/common/domain"

type SportLineQueryService interface {
	GetLinesBySportTypes(sportTypes []domain.SportType) ([]*domain.SportLine, error)
}
