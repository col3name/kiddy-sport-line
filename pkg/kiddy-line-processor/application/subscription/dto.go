package subscription

import (
	"github.com/col3name/lines/pkg/common/domain"
)

type MessageToSubscribeDTO struct {
	ClientId             int
	Sports               []domain.SportType
	UpdateIntervalSecond int32
}
