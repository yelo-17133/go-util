package timeUtil

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	for _, v := range []interface{}{
		1563362142,
		1563362142564,
		"now",
		"today",
		"tommorow",
		"today,2hour",
		"today,-2hour",
		"today,+2hour",
		"today,2 hour",
		"today,+2 hour",
		"tommorow,-1ns",
		"2019-07-17 19:15:42",
		"2019-07-17 19:15",
		"2019-07-17",
		"07-17 19:15",
	} {
		if val, err := Parse(fmt.Sprint(v)); err != nil {
			t.Error(err)
		} else {
			t.Logf("%v: %v", v, val.Format(GeneralFormatNano))
		}
	}
}

func TestParseOne(t *testing.T) {
	str := "today,2hour"
	if val, err := Parse(str); err != nil {
		t.Error(err)
	} else {
		t.Logf("%v: %v", str, val.Format(GeneralFormatNano))
	}
}
