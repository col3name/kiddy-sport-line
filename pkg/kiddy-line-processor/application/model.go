package application

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
)

type SubscriptionMessageDTO struct {
	ClientId             int
	Sports               []commonDomain.SportType
	UpdateIntervalSecond int32
}

type Sport struct {
	Type string
	Line float32
}
