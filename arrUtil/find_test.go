package arrUtil

import (
	"fmt"
	"testing"
)

func TestFindInterface(t *testing.T) {
	type abc struct{ Id int }
	arr := []*abc{
		{Id: 1},
		{Id: 3},
		{Id: 5},
		{Id: 7},
	}

	resultArr, ok := Find(arr, func(i int, a interface{}) bool {
		return a.(*abc).Id == 3
	}, -1).([]*abc)
	if !ok || len(resultArr) != 1 || resultArr[0].Id != 3 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(*abc).Id > 1
	}, -1).([]*abc)
	if !ok || len(resultArr) != 3 || resultArr[0].Id != 3 || resultArr[1].Id != 5 || resultArr[2].Id != 7 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(*abc).Id == 123456576
	}, -1).([]*abc)
	if !ok || len(resultArr) != 0 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal := FindOne(arr, func(i int, a interface{}) bool {
		return a.(*abc).Id > 1
	})
	if resultVal == nil || resultVal.(*abc).Id != 3 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal = FindOne(arr, func(i int, a interface{}) bool {
		return a.(*abc).Id < 1
	})
	if resultVal != nil {
		t.Error(fmt.Sprintf("assert faild"))
	}
}

func TestFindInt(t *testing.T) {
	arr := []int{1, 3, 5, 7}

	resultArr, ok := Find(arr, func(i int, a interface{}) bool {
		return a.(int) == 3
	}, -1).([]int)
	if !ok || len(resultArr) != 1 || resultArr[0] != 3 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(int) > 1
	}, -1).([]int)
	if !ok || len(resultArr) != 3 || resultArr[0] != 3 || resultArr[1] != 5 || resultArr[2] != 7 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(int) == 123456576
	}, -1).([]int)
	if !ok || len(resultArr) != 0 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal := FindOne(arr, func(i int, a interface{}) bool {
		return a.(int) > 1
	})
	if resultVal != 3 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal = FindOne(arr, func(i int, a interface{}) bool {
		return a.(int) < 1
	})
	if resultVal != nil {
		t.Error(fmt.Sprintf("assert faild"))
	}
}

func TestFindString(t *testing.T) {
	arr := []string{"1", "3", "5", "7"}

	resultArr, ok := Find(arr, func(i int, a interface{}) bool {
		return a.(string) == "3"
	}, -1).([]string)
	if !ok || len(resultArr) != 1 || resultArr[0] != "3" {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(string) > "1"
	}, -1).([]string)
	if !ok || len(resultArr) != 3 || resultArr[0] != "3" || resultArr[1] != "5" || resultArr[2] != "7" {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultArr, ok = Find(arr, func(i int, a interface{}) bool {
		return a.(string) == "123456576"
	}, -1).([]string)
	if !ok || len(resultArr) != 0 {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal := FindOne(arr, func(i int, a interface{}) bool {
		return a.(string) > "1"
	})
	if resultVal != "3" {
		t.Error(fmt.Sprintf("assert faild"))
	}

	resultVal = FindOne(arr, func(i int, a interface{}) bool {
		return a.(string) < "1"
	})
	if resultVal != nil {
		t.Error(fmt.Sprintf("assert faild"))
	}
}

func TestFindAll(t *testing.T) {
	arr := make([]string, 0, 8)
	arr = append(arr, "a", "b", "c")

	result := FindAll(arr, func(i int, a interface{}) bool {
		return a != "a"
	}).([]string)
	if n := len(result); n != 2 {
		t.Errorf("assert faild: %v", n)
	} else {
		if result[0] != "b" {
			t.Errorf("assert faild: %v", result[0])
		}
		if result[1] != "c" {
			t.Errorf("assert faild: %v", result[1])
		}
	}

	result = FindAll(arr, func(i int, a interface{}) bool {
		return a == "not exist"
	}).([]string)
	if n := len(result); n != 0 {
		t.Errorf("assert faild: %v", n)
	}
}
