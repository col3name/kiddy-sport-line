package times

import (
	"time"
)

type Ticker interface {
	Handle(seconds int32, fn func()) *time.Ticker
}

type TimeTicker struct {
}

func NewTimeTicker() *TimeTicker {
	return &TimeTicker{}
}

func (TimeTicker) Handle(seconds int32, fn func()) *time.Ticker {
	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	go func() {
		for range ticker.C {
			fn()
		}
	}()
	return ticker
}
