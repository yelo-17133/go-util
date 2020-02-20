package arrUtil

import (
	"strconv"
	"yelo/go-util/convertor"
)

// ------------------------------------------------------------------------------ number to number
func IntToInt64(a []int) []int64 {
	if a == nil {
		return nil
	}

	arr := make([]int64, len(a))
	for i, v := range a {
		arr[i] = int64(v)
	}
	return arr
}

func Int64ToInt(a []int64) []int {
	if a == nil {
		return nil
	}

	arr := make([]int, len(a))
	for i, v := range a {
		arr[i] = int(v)
	}
	return arr
}

// ------------------------------------------------------------------------------ xxx to string
func Int64ToStr(a []int64, f ...func(a int64) string) []string {
	if a == nil {
		return nil
	}

	var theF func(n int64) string
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = func(n int64) string {
			return strconv.FormatInt(n, 10)
		}
	}

	arr := make([]string, len(a))
	for i, v := range a {
		arr[i] = theF(v)
	}
	return arr
}

func IntToStr(a []int, f ...func(a int) string) []string {
	if a == nil {
		return nil
	}

	var theF func(n int) string
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = func(n int) string {
			return strconv.FormatInt(int64(n), 10)
		}
	}

	arr := make([]string, len(a))
	for i, v := range a {
		arr[i] = theF(v)
	}
	return arr
}

func ObjToStr(a []interface{}, f ...func(in interface{}) string) []string {
	if a == nil {
		return nil
	}

	var theF func(in interface{}) string
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = convertor.ToStringNoError
	}

	arr := make([]string, len(a))
	for i, v := range a {
		arr[i] = theF(v)
	}
	return arr
}

// ------------------------------------------------------------------------------ string to xxx
func StrToObj(a []string) []interface{} {
	if a == nil {
		return nil
	}

	arr := make([]interface{}, len(a))
	for i, v := range a {
		arr[i] = v
	}
	return arr
}

func StrToInt64(a []string, f ...func(string) (int64, error)) ([]int64, error) {
	if a == nil {
		return nil, nil
	}

	var theF func(string) (int64, error)
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = func(s string) (i int64, e error) {
			return convertor.ToInt64(s)
		}
	}

	arr := make([]int64, len(a))
	for i, v := range a {
		if n, err := theF(v); err != nil {
			return arr, err
		} else {
			arr[i] = n
		}
	}
	return arr, nil
}

func StrToInt64NoError(a []string, f ...func(string) (int64, error)) []int64 {
	v, _ := StrToInt64(a, f...)
	return v
}

func StrToInt(a []string, f ...func(string) (int, error)) ([]int, error) {
	if a == nil {
		return nil, nil
	}

	var theF func(string) (int, error)
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = func(s string) (i int, e error) {
			return convertor.ToInt(s)
		}
	}

	arr := make([]int, len(a))
	for i, v := range a {
		if n, err := theF(v); err != nil {
			return arr, err
		} else {
			arr[i] = n
		}
	}
	return arr, nil
}

func StrToIntNoError(a []string, f ...func(string) (int, error)) []int {
	v, _ := StrToInt(a, f...)
	return v
}
