package mapUtil

import (
	"fmt"
	"reflect"
)

// 判断两个或多个 map 是否相等
// src 和 dest 的类型必须相同
func Equal(src interface{}, dest ...interface{}) bool {
	return doEqual(nil, src, dest...)
}

// 判断两个或多个 map 是否相等
func doEqual(equalFunc func(a, b interface{}) bool, src interface{}, dest ...interface{}) bool {
	// 确认参数类型相同，并取出所有的 map
	types := make([]reflect.Type, 0, len(dest)+1)
	if src != nil {
		types = append(types, reflect.TypeOf(src))
	}
	for _, v := range dest {
		if v != nil {
			types = append(types, reflect.TypeOf(v))
		}
	}
	mapCount := len(types)
	if mapCount == 0 {
		return true
	}
	for i := 1; i < mapCount; i++ {
		if types[0] != types[i] {
			panic(fmt.Sprintf("参数类型不同 (%v, %v)", types[0], types[i]))
		}
	}

	isPtrType := types[0].Kind() == reflect.Ptr

	// 获取参数对应的 map 对象（处理指向 map 的指针），并判断类型，将所有要比较的 map 放在一个数组中
	maps := make([]reflect.Value, 0, len(dest))
	if src != nil {
		if isPtrType {
			maps = append(maps, reflect.ValueOf(src).Elem())
		} else {
			maps = append(maps, reflect.ValueOf(src))
		}
	}
	for _, v := range dest {
		if v != nil {
			if isPtrType {
				maps = append(maps, reflect.ValueOf(v).Elem())
			} else {
				maps = append(maps, reflect.ValueOf(v))
			}
		}
	}
	if maps[0].Kind() != reflect.Map {
		panic(fmt.Sprintf("参数不是 map 类型 (%v)", types[0]))
	}

	// 遍历 maps 数组，验证两两相等
	keys := maps[0].MapKeys()
	for i := 1; i < mapCount; i++ {
		if !doEqualCallback(equalFunc, keys, maps[0], maps[i]) {
			return false
		}
	}

	return true
}

// 比较两个 map 是否相等
func doEqualCallback(equalFunc func(a, b interface{}) bool, keys []reflect.Value, a, b reflect.Value) bool {
	for _, key := range keys {
		destVal := b.MapIndex(key)
		if !destVal.IsValid() {
			return false
		} else if equalFunc == nil {
			if a.MapIndex(key).Interface() != destVal.Interface() {
				return false
			}
		} else {
			if !equalFunc(a.MapIndex(key).Interface(), destVal.Interface()) {
				return false
			}
		}
	}

	for _, key := range b.MapKeys() {
		srcVal := a.MapIndex(key)
		if !srcVal.IsValid() {
			return false
		} else if equalFunc == nil {
			if b.MapIndex(key).Interface() != srcVal.Interface() {
				return false
			}
		} else {
			if !equalFunc(b.MapIndex(key).Interface(), srcVal.Interface()) {
				return false
			}
		}
	}

	return true
}
