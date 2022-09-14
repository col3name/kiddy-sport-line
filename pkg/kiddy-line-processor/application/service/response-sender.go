package service

import (
	"github.com/col3name/lines/pkg/common/domain"
)

type ResponseSenderService interface {
	Send(sports []*domain.SportLine) error
}
