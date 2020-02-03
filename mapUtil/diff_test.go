package mapUtil

import (
	"fmt"
	"testing"
)

type strIntMap map[string]int

func TestDiffEmptyOfError(t *testing.T) {
	var diff interface{}
	var err error

	//
	diff = Diff(nil, nil)
	if diff != nil {
		t.Errorf("assert faild")
	}
	diff = Diff(nil, strIntMap{})
	if diff == nil {
		t.Errorf("assert faild")
	}
	diff = Diff(map[string]string{}, nil)
	if diff == nil {
		t.Errorf("assert faild")
	}

	// 类型不同，需要报错
	_, err = testDiff(map[string]string{}, strIntMap{})
	if err == nil {
		t.Errorf("类型不同，应该报错而未报")
	}

	// 虽然实际上是同一个类型，但仍然需要报错
	_, err = testDiff(map[string]int{}, strIntMap{})
	if err == nil {
		t.Errorf("类型不同，应该报错而未报")
	}

	// 同一个类型，一个是结构体一个是指针
	_, err = testDiff(&strIntMap{}, strIntMap{})
	if err == nil {
		t.Errorf("类型不同，应该报错而未报")
	}
}

func TestDiffType1(t *testing.T) {
	diff := Diff(strIntMap{}, strIntMap{})
	_, ok := diff.(strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
	}
}

func TestDiffType2(t *testing.T) {
	diff := Diff(&strIntMap{}, &strIntMap{})
	_, ok := diff.(*strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
	}
}

func TestDiffType3(t *testing.T) {
	diff := Diff(strIntMap{"a": 1, "b": 2, "c": 3}, strIntMap{"a": 1, "b": 222})
	dict, ok := diff.(strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
		return
	}
	if len(dict) != 2 {
		t.Errorf("assert faild: %v", len(dict))
	}
	if dict["b"] != 222 {
		t.Errorf("assert faild: %v", dict["b"])
	}
	if dict["c"] != 0 {
		t.Errorf("assert faild: %v", dict["c"])
	}
}

func TestDiffType4(t *testing.T) {
	diff := Diff(nil, strIntMap{"a": 1, "b": 2, "c": 3}, strIntMap{"a": 1, "b": 222})
	dict, ok := diff.(strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
		return
	}
	if len(dict) != 3 {
		t.Errorf("assert faild: %v", len(dict))
	}
	if dict["a"] != 1 {
		t.Errorf("assert faild: %v", dict["b"])
	}
	if dict["b"] != 222 {
		t.Errorf("assert faild: %v", dict["b"])
	}
	if dict["c"] != 3 {
		t.Errorf("assert faild: %v", dict["c"])
	}
}

func TestDiffType5(t *testing.T) {
	diff := Diff(&strIntMap{"a": 1, "b": 2, "c": 3}, &strIntMap{"a": 1, "b": 222})
	dict, ok := diff.(*strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
		return
	}
	if len(*dict) != 2 {
		t.Errorf("assert faild: %v", len(*dict))
	}
	if (*dict)["b"] != 222 {
		t.Errorf("assert faild: %v", (*dict)["b"])
	}
	if (*dict)["c"] != 0 {
		t.Errorf("assert faild: %v", (*dict)["c"])
	}
}

func TestDiffType6(t *testing.T) {
	diff := Diff(nil, &strIntMap{"a": 1, "b": 2, "c": 3}, &strIntMap{"a": 1, "b": 222})
	dict, ok := diff.(*strIntMap)
	if !ok {
		t.Errorf("类型转换出错: %v", diff)
		return
	}
	if len(*dict) != 3 {
		t.Errorf("assert faild: %v", len(*dict))
	}
	if (*dict)["a"] != 1 {
		t.Errorf("assert faild: %v", (*dict)["b"])
	}
	if (*dict)["b"] != 222 {
		t.Errorf("assert faild: %v", (*dict)["b"])
	}
	if (*dict)["c"] != 3 {
		t.Errorf("assert faild: %v", (*dict)["c"])
	}
}

func testDiff(a, b interface{}) (v interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	return Diff(a, b), nil
}
