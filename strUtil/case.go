package strUtil

import (
	"regexp"
	"strings"
)

func UcFirst(s string) string {
	n := len(s)
	if n == 0 {
		return s
	} else if n == 1 {
		return strings.ToUpper(s)
	} else {
		c := s[:1]
		if u := strings.ToUpper(c); u == c {
			return s
		} else {
			return u + s[1:]
		}
	}
}

func LcFirst(s string) string {
	n := len(s)
	if n == 0 {
		return s
	} else if n == 1 {
		return strings.ToLower(s)
	} else {
		c := s[:1]
		if u := strings.ToLower(c); u == c {
			return s
		} else {
			return u + s[1:]
		}
	}
}

// 将 camel 命名法转换为大写中划线分割格式（Http Header 格式）
func CamelToUpperKebab(s string) string {
	for _, ss := range camelToKebabPattern.FindAllString(s, -1) {
		s = strings.Replace(s, ss, ss[:1]+"-"+ss[1:], -1)
	}
	return UcFirst(s)
}

// 将 camel 命名法转换为小写中划线分割格式
func CamelToLowerKebab(s string) string {
	for _, ss := range camelToKebabPattern.FindAllString(s, -1) {
		s = strings.Replace(s, ss, ss[:1]+"-"+ss[1:], -1)
	}
	return strings.ToLower(s)
}

var camelToKebabPattern = regexp.MustCompile("[a-z][A-Z]")

func KebabToCamel(s string) string {
	arr := strings.Split(s, "-")
	for i, s := range arr {
		if i == 0 {
			arr[i] = LcFirst(s)
		} else {
			arr[i] = UcFirst(s)
		}
	}
	return strings.Join(arr, "")
}

func KebabToSnake(s string) string {
	return strings.ToLower(strings.Replace(s, "-", "_", -1))
}

// 下划线小写格式转换为 Camel 格式
func SnakeToCamel(s string) string {
	if s != "" {
		arr := Map(strings.Split(s, "_"), func(i int, s string) string {
			if i == 0 {
				return LcFirst(s)
			} else {
				return UcFirst(s)
			}
		})
		return strings.Join(arr, "")
	}
	return ""
}

// 下划线小写格式转换为 Pascal 格式
func SnakeToPascal(s string) string {
	if s != "" {
		arr := Map(strings.Split(s, "_"), func(_ int, s string) string {
			return UcFirst(s)
		})
		return strings.Join(arr, "")
	}
	return ""
}

func SnakeToUpperKebab(s string) string {
	if s != "" {
		arr := Map(strings.Split(s, "_"), func(_ int, s string) string {
			return UcFirst(s)
		})
		return strings.Join(arr, "-")
	}
	return ""
}
