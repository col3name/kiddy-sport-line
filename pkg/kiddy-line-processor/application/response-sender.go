package application

import (
	"github.com/col3name/lines/pkg/common/domain"
)

type responseSender interface {
	Send(sports []*domain.SportLine) error
}
