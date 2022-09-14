package model

import (
	commonDomain "github.com/col3name/lines/pkg/common/domain"
	"time"
)

type SportTypeMap map[commonDomain.SportType]float32

type ClientSubscription struct {
	Sports SportTypeMap
	Task   *time.Ticker
}
