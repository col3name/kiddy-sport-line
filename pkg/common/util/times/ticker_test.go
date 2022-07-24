package times

import (
	"fmt"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	ticker := TickerHandle(1, func() {
		fmt.Println(time.Now().Second())
	})
	time.Sleep(10 * time.Second)
	ticker.Stop()
}
