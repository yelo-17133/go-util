package strUtil

import (
	"fmt"
	"go-util/jsonUtil"
	"testing"
)

func TestSplitToInt(t *testing.T) {
	var arr []int
	var err error

	arr, err = SplitToInt("", ",", true)
	if err != nil {
		t.Error(fmt.Errorf("error occured: %v", err))
	} else if len(arr) != 0 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	arr, err = SplitToInt("1,2,,3,", ",", true)
	if err != nil {
		t.Error(fmt.Errorf("error occured: %v", err))
	} else if len(arr) != 3 || arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
		t.Error(fmt.Sprintf("assert faild: %v", jsonUtil.MustMarshalToString(arr)))
	}
}
