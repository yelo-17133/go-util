package timeUtil

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestAbc(t *testing.T) {
	now := time.Date(1980, 1, 1, 0, 0, 0, 0, time.Local)
	t.Log(math.MaxInt32, now.Unix(), ToMs(now))
}

func TestMax(t *testing.T) {
	now := time.Now()
	val := Max(now.Add(-time.Minute), now, now.Add(time.Minute))
	if val != now.Add(time.Minute) {
		t.Errorf("assert faild: %v, now=%v", val, now)
	}
}

func TestMin(t *testing.T) {
	now := time.Now()
	val := Min(now.Add(-time.Minute), now, now.Add(time.Minute))
	if val != now.Add(-time.Minute) {
		t.Errorf("assert faild: %v, now=%v", val, now)
	}
}

func TestVar(t *testing.T) {
	var str string

	str = Local2000.Format("2006-01-02 15:04:05.000000000")
	if str != "2000-01-01 00:00:00.000000000" {
		t.Error(fmt.Sprintf("assert faild: %v", str))
	}

	str = EndOf2037.Format("2006-01-02 15:04:05.000000000")
	if str != "2037-12-31 23:59:59.999999999" {
		t.Error(fmt.Sprintf("assert faild: %v", str))
	}
}

func _TestToSecondStr(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(ToSecondStr(time.Now(), 6))
		time.Sleep(time.Millisecond * 50)
	}
}

func _TestToMsStr(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(ToMsStr(time.Now(), 6))
		time.Sleep(time.Microsecond * 50)
	}
}

func TestGeneralParse(t *testing.T) {
	Parse("2019-6-4 12:35:36.111112")

	for _, str := range []string{
		"6-4",
		"6-04",
		"06-4",
		"06-04",
		"2019-6-4",
		"2019-06-4",
		"2019-6-04",
		"2019-06-04",
		"2019-6-4 12:13",
		"2019-6-4 12:13:36",
		"2019-6-4 12:35:36.111112000000000",
	} {
		val, err := Parse(str)
		if err != nil {
			t.Errorf("error: %v", err)
		} else {
			t.Logf("%s --> %s", str, val.Format(GeneralFormatNano))
		}
	}
}
