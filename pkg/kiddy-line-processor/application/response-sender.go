package application

import (
	"github.com/col3name/lines/pkg/kiddy-line-processor/domain"
)

type responseSender interface {
	Send(sports []*domain.Sport) error
}
