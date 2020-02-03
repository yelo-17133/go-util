package strUtil

import (
	"fmt"
	"strconv"
	"strings"
)

func ToInt64Array(a []string, ignoreEmpty bool) ([]int64, error) {
	if a == nil {
		return nil, nil
	}
	b := make([]int64, 0, len(a))
	for _, s := range a {
		if s = strings.TrimSpace(s); s != "" {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无法将 %v 转换为 int64", s)
			}
			b = append(b, n)
		} else if !ignoreEmpty {
			b = append(b, 0)
		}
	}
	return b, nil
}

func ToIntArray(a []string, ignoreEmpty bool) ([]int, error) {
	if a == nil {
		return nil, nil
	}
	b := make([]int, 0, len(a))
	for _, s := range a {
		if s = strings.TrimSpace(s); s != "" {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无法将 %v 转换为 int", s)
			}
			b = append(b, int(n))
		} else if !ignoreEmpty {
			b = append(b, 0)
		}
	}
	return b, nil
}

func ToInterfaceArray(a []string) []interface{} {
	if a != nil {
		arr := make([]interface{}, len(a))
		for i, v := range a {
			arr[i] = v
		}
		return arr
	}
	return nil
}
