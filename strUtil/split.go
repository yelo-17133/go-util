package strUtil

import (
	"fmt"
	"regexp"
	"strings"
	"yelo/go-util/arrUtil"
)

func Split(s, sep string, ignoreEmpty bool, f ...func(string) string) []string {
	var theF func(string) string
	if len(f) != 0 && f[0] != nil {
		theF = f[0]
	} else {
		theF = func(s string) string {
			return s
		}
	}

	arr := strings.Split(s, sep)
	result := make([]string, 0, len(arr))
	for _, str := range arr {
		str = theF(str)
		if !ignoreEmpty || str != "" {
			result = append(result, str)
		}
	}
	return result
}

func SplitToInt64(s, sep string, ignoreEmpty bool) ([]int64, error) {
	return arrUtil.StrToInt64(Split(s, sep, ignoreEmpty))
}

func SplitToInt64NoError(s, sep string, ignoreEmpty bool) []int64 {
	v, _ := arrUtil.StrToInt64(Split(s, sep, ignoreEmpty))
	return v
}

func SplitToInt(s, sep string, ignoreEmpty bool) ([]int, error) {
	return arrUtil.StrToInt(Split(s, sep, ignoreEmpty))
}

func SplitToIntNoError(s, sep string, ignoreEmpty bool) []int {
	v, _ := arrUtil.StrToInt(Split(s, sep, ignoreEmpty))
	return v
}

func PregSplitToInt64(s, pattern string) ([]int64, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式语法不正确: %v, pattern=%s", err, pattern)
	}
	return arrUtil.StrToInt64(reg.Split(s, -1))
}

func PregSplitToInt64NoError(s, pattern string) []int64 {
	v, _ := PregSplitToInt64(s, pattern)
	return v
}

func PregSplitToInt(s, pattern string) ([]int, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式语法不正确: %v, pattern=%s", err, pattern)
	}
	return arrUtil.StrToInt(reg.Split(s, -1))
}

func PregSplitToIntNoError(s, pattern string) []int {
	v, _ := PregSplitToInt(s, pattern)
	return v
}
