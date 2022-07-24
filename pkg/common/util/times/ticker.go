package times

import (
	"time"
)

func TickerHandle(seconds int32, fn func()) *time.Ticker {
	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	go func() {
		for range ticker.C {
			fn()
		}
	}()
	return ticker
}
