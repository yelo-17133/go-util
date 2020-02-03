package arrUtil

import (
	"testing"
)

func TestSlice(t *testing.T) {
	rawArr := []int{1, 2, 3, 4, 5}

	if arr := rawArr[0:0]; len(arr) != 0 {
		t.Error("assert faild")
	}

	if arr := rawArr[1:1]; len(arr) != 0 {
		t.Error("assert faild")
	}

	if arr := rawArr[4:4]; len(arr) != 0 {
		t.Error("assert faild")
	}

	if arr := rawArr[5:5]; len(arr) != 0 {
		t.Error("assert faild")
	}

	if arr := rawArr[1:2]; len(arr) != 1 || arr[0] != 2 {
		t.Error("assert faild")
	}

	if arr := rawArr[2:4]; len(arr) != 2 || arr[0] != 3 || arr[1] != 4 {
		t.Error("assert faild")
	}

	if arr := append(rawArr[0:0], 1); len(arr) != 1 || arr[0] != 1 {
		t.Error("assert faild")
	}

	if arr := append(rawArr[4:4], 1); len(arr) != 1 || arr[0] != 1 {
		t.Error("assert faild")
	}

	if arr := append(rawArr[0:0], rawArr[4:4]...); len(arr) != 0 {
		t.Error("assert faild")
	}
}

func TestSliceModify(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i]%3 == 0 {
			arr = append(arr[:i], arr[i+1:]...)
		}
	}
	if len(arr) != 6 {
		t.Error("assert faild")
	}
	if arr[0] != 1 || arr[1] != 2 || arr[2] != 4 || arr[3] != 5 || arr[4] != 7 || arr[5] != 8 {
		t.Error("assert faild")
	}
}
