package strUtil

import (
	"fmt"
	"regexp"
	"strings"
)

func Split(s, sep string, ignoreEmpty bool, f func(string) string) []string {
	arr := strings.Split(s, sep)
	result := make([]string, 0, len(arr))
	for _, str := range arr {
		if f != nil {
			str = f(str)
		}
		if !ignoreEmpty || str != "" {
			result = append(result, str)
		}
	}
	return result
}

func SplitToInt64(s, sep string, ignoreEmpty bool) ([]int64, error) {
	return ToInt64Array(strings.Split(s, sep), ignoreEmpty)
}

func MustSplitToInt64(s, sep string, ignoreEmpty bool) []int64 {
	v, _ := ToInt64Array(strings.Split(s, sep), ignoreEmpty)
	return v
}

func SplitToInt(s, sep string, ignoreEmpty bool) ([]int, error) {
	return ToIntArray(strings.Split(s, sep), ignoreEmpty)
}

func MustSplitToInt(s, sep string, ignoreEmpty bool) []int {
	v, _ := ToIntArray(strings.Split(s, sep), ignoreEmpty)
	return v
}

func PregSplitToInt64(s, pattern string, ignoreEmpty bool) ([]int64, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式语法不正确: %v, pattern=%s", err, pattern)
	}
	return ToInt64Array(reg.Split(s, -1), ignoreEmpty)
}

func MustPregSplitToInt64(s, pattern string, ignoreEmpty bool) []int64 {
	v, _ := PregSplitToInt64(s, pattern, ignoreEmpty)
	return v
}

func PregSplitToInt(s, pattern string, ignoreEmpty bool) ([]int, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式语法不正确: %v, pattern=%s", err, pattern)
	}
	return ToIntArray(reg.Split(s, -1), ignoreEmpty)
}

func MustPregSplitToInt(s, pattern string, ignoreEmpty bool) []int {
	v, _ := PregSplitToInt(s, pattern, ignoreEmpty)
	return v
}
