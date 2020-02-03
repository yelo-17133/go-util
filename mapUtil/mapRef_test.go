package mapUtil

import "testing"

// 验证 map 是引用类型：
// 1、函数内部对 map 的改动在函数外面是可以拿到新值的
func TestMapRef(t *testing.T) {
	func1 := func(a map[string]int) {
		if a == nil {
			a = make(map[string]int)
		}
		a["a"] = 1
		a["b"] = 2
		a["c"] = 3
	}

	dict1 := make(map[string]int)
	func1(dict1)
	a, b, c := dict1["a"], dict1["b"], dict1["c"]
	if a != 1 {
		t.Errorf("assert faild")
	}
	if b != 2 {
		t.Errorf("assert faild")
	}
	if c != 3 {
		t.Errorf("assert faild")
	}

	var dict2 map[string]int
	func1(dict2)
	if dict2 != nil {
		t.Errorf("assert faild")
	}
}
