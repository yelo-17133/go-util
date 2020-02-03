package timeRoundedCounter

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	counter := New(time.Minute, 10).(*counterImpl)
	now := time.Now()
	counter.countNow(7, true, now.Add(-5*time.Minute))
	counter.countNow(5, true, now.Add(-2*time.Minute))
	counter.countNow(2, true, now.Add(-1*time.Minute))
	counter.countNow(3, true, now)
	data := counter.getDataSliceNow(0, 10, false, now)
	dataLen := len(data)
	if dataLen != 10 {
		t.Errorf("assert faild: %v", dataLen)
	}
	data = counter.getDataSliceNow(0, 100, false, now)
	dataLen = len(data)
	if dataLen != 10 {
		t.Errorf("assert faild: %v", dataLen)
	}
	data = counter.getDataSliceNow(5, 100, false, now)
	dataLen = len(data)
	if dataLen != 5 {
		t.Errorf("assert faild: %v", dataLen)
	}
	data = counter.getDataSliceNow(0, 100, true, now)
	dataLen = len(data)
	if dataLen != 6 {
		t.Errorf("assert faild: %v", dataLen)
	} else {
		if data[0] != 3 {
			t.Errorf("assert faild")
		}
		if data[1] != 2 {
			t.Errorf("assert faild")
		}
		if data[2] != 5 {
			t.Errorf("assert faild")
		}
		if data[3] != 0 {
			t.Errorf("assert faild")
		}
		if data[4] != 0 {
			t.Errorf("assert faild")
		}
		if data[5] != 7 {
			t.Errorf("assert faild")
		}
	}
	data = counter.getDataSliceNow(2, 100, true, now)
	dataLen = len(data)
	if dataLen != 4 {
		t.Errorf("assert faild: %v", dataLen)
	} else {
		if data[0] != 5 {
			t.Errorf("assert faild")
		}
		if data[1] != 0 {
			t.Errorf("assert faild")
		}
		if data[2] != 0 {
			t.Errorf("assert faild")
		}
		if data[3] != 7 {
			t.Errorf("assert faild")
		}
	}
}
