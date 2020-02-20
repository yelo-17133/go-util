package arrUtil

import (
	"testing"
	"yelo/go-util/mathUtil"
)

func TestStrToInt64(t *testing.T) {
	arr, err := StrToInt64([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
		return
	}
	if n := mathUtil.SumInt64(arr); n != 6 {
		t.Errorf("assert faild: %v", n)
		return
	}
}
