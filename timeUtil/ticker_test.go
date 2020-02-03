package timeUtil

import (
	"fmt"
	"testing"
	"time"
)

func TestTickerImpl_Stop(t *testing.T) {
	ticker := NewTicker(300*time.Millisecond, time.Second, func() {
		fmt.Println(time.Now().Format(GeneralFormatNano))
	})
	time.Sleep(5 * time.Second)
	ticker.SetDuration(500 * time.Millisecond)
	time.Sleep(5 * time.Second)
	val := ticker.Stop(time.Second)
	if !val {
		t.Error("stop error")
	}
}
