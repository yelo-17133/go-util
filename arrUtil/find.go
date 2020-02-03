package arrUtil

import "reflect"

// 从数组中查找符合条件的元素，并返回包含这些元素的新数组
//   a: 要查找的数组
//   match: 匹配函数
//   count: 最多返回的数量，<0 表示全部
func Find(a interface{}, match func(i int, a interface{}) bool, count int) interface{} {
	if v := doFind(a, match, count); v != nil {
		return v.Interface()
	}
	return nil
}

// 从数组中查找符合条件的元素，并返回包含这些元素的新数组
//   a: 要查找的数组
//   match: 匹配函数
func FindAll(a interface{}, match func(i int, a interface{}) bool) interface{} {
	if v := doFind(a, match, -1); v != nil {
		return v.Interface()
	}
	return nil
}

// 从数组中查找第一个符合条件的元素，并返回该元素
//   a: 要查找的数组
//   match: 匹配函数
func FindOne(a interface{}, match func(i int, a interface{}) bool) interface{} {
	if v := doFind(a, match, 1); v != nil && v.Len() != 0 {
		return v.Index(0).Interface()
	}
	return nil
}

func doFind(a interface{}, match func(i int, a interface{}) bool, count int) *reflect.Value {
	if a == nil || count == 0 {
		return nil
	}

	reflectVal := reflect.ValueOf(a)
	if kind := reflectVal.Kind(); kind != reflect.Array && kind != reflect.Slice {
		panic("参数 a 必须是数组或切片")
	}

	maxLen, realLen := reflectVal.Len(), 0
	resultItems := reflect.MakeSlice(reflectVal.Type(), 0, maxLen)
	for i := 0; i < maxLen; i++ {
		v := reflectVal.Index(i)
		if match == nil || match(i, v.Interface()) {
			resultItems = reflect.Append(resultItems, v)
			if realLen = realLen + 1; count > 0 && realLen == count {
				break
			}
		}
	}

	result := reflect.New(reflectVal.Type()).Elem()
	result.Set(resultItems)

	return &result
}
