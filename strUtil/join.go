package strUtil

import (
	"reflect"
	"strings"
	"yelo/go-util/convertor"
)

func JoinInt64(a []int64, sep string) string {
	return Join(a, sep, nil)
}

func JoinInt32(a []int32, sep string) string {
	return Join(a, sep, nil)
}

func JoinInt(a []int, sep string) string {
	return Join(a, sep, nil)
}

func Join(a interface{}, sep string, f func(i int, a interface{}) string) string {
	if a == nil {
		return ""
	}

	reflectVal := reflect.ValueOf(a)
	if kind := reflectVal.Kind(); kind != reflect.Array && kind != reflect.Slice {
		panic("参数 a 必须是数组或切片")
	}

	valLen := reflectVal.Len()
	arr := make([]string, valLen)
	for i := 0; i < valLen; i++ {
		if f == nil {
			arr[i] = convertor.ToStringNoError(reflectVal.Index(i).Interface())
		} else {
			arr[i] = f(i, reflectVal.Index(i).Interface())
		}
	}
	return strings.Join(arr, sep)
}
