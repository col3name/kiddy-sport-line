package repo

import (
	"github.com/col3name/lines/pkg/common/domain"
)

type SportLineRepo interface {
	Store(model *domain.SportLine) error
}
