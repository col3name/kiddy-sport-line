package application

import (
	"github.com/col3name/lines/pkg/common/domain"
)

type SubscriptionMessageDTO struct {
	ClientId             int
	Sports               []domain.SportType
	UpdateIntervalSecond int32
}
