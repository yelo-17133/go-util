package strUtil

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"net/url"
	"strings"
)

// ------------------------------------------------------------------------------ encode & decode
func UrlDecode(s string) string {
	v, _ := url.QueryUnescape(s)
	return v
}

func Sha1(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

func Md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// ------------------------------------------------------------------------------
func ReplaceMap(str string, replace map[string]string) string {
	for key, val := range replace {
		str = strings.Replace(str, key, val, -1)
	}
	return str
}

func For(in []string, f func(int, string)) {
	if in == nil || f == nil {
		return
	}
	for i, s := range in {
		f(i, s)
	}
}

func Map(in []string, f func(int, string) string) []string {
	if in == nil || f == nil {
		return in
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = f(i, s)
	}
	return out
}

func MapObj(in []interface{}, f func(i int, a interface{}) string) []string {
	if in == nil {
		return nil
	} else if f == nil {
		f = func(i int, a interface{}) string {
			return fmt.Sprint(a)
		}
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = f(i, s)
	}
	return out
}
