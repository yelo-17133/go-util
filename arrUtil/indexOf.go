package arrUtil

import (
	"reflect"
	"sort"
	"strings"
)

// 获取指定值在数组中的索引。-1 表示不在数组中
func IndexOfInt(a []int, val int) int {
	for i, n := range a {
		if n == val {
			return i
		}
	}
	return -1
}

// 获取指定值在数组中的索引。-1 表示不在数组中
func IndexOfInt64(a []int64, val int64) int {
	for i, n := range a {
		if n == val {
			return i
		}
	}
	return -1
}

// 获取指定值在数组中的索引。-1 表示不在数组中
func IndexOfString(a []string, val string, ignoreCase bool) int {
	if ignoreCase {
		for i, s := range a {
			if strings.EqualFold(s, val) {
				return i
			}
		}
	} else {
		for i, s := range a {
			if s == val {
				return i
			}
		}
	}
	return -1
}

// 获取指定值在有序数组中的索引。-1 表示不在数组中。数组必须是升序排序的
func IndexOfSortedInt(a []int, val int) int {
	count := len(a)
	pos := sort.Search(count, func(i int) bool { return a[i] >= val })
	if pos >= count {
		return -1
	} else if v := a[pos]; v != val {
		return -1
	} else {
		return pos
	}
}

// 获取指定值在有序数组中的索引。-1 表示不在数组中。数组必须是升序排序的
func IndexOfSortedInt64(a []int64, val int64) int {
	count := len(a)
	pos := sort.Search(count, func(i int) bool { return a[i] >= val })
	if pos >= count {
		return -1
	} else if v := a[pos]; v != val {
		return -1
	} else {
		return pos
	}
}

func IndexOf(a interface{}, match func(a interface{}) bool) int {
	if a == nil {
		return -1
	}

	reflectVal := reflect.ValueOf(a)
	if kind := reflectVal.Kind(); kind != reflect.Array && kind != reflect.Slice {
		panic("参数 a 必须是数组或切片")
	}

	for i, n := 0, reflectVal.Len(); i < n; i++ {
		v := reflectVal.Index(i)
		if match == nil || match(v.Interface()) {
			return i
		}
	}

	return -1
}
