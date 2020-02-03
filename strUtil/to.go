package strUtil

import (
	"fmt"
	"strconv"
)

func MustToInt64(s string, a ...interface{}) int64 {
	if len(a) != 0 {
		s = fmt.Sprintf(s, a...)
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func MustToInt32(s string, a ...interface{}) int32 {
	return int32(MustToInt64(s, a...))
}

func MustToInt(s string, a ...interface{}) int {
	return int(MustToInt64(s, a...))
}

func MustToBool(s string, a ...interface{}) bool {
	if len(a) != 0 {
		s = fmt.Sprintf(s, a...)
	}
	v, e := strconv.ParseBool(s)
	return e == nil && v
}

func MustToFloat64(s string, a ...interface{}) float64 {
	if len(a) != 0 {
		s = fmt.Sprintf(s, a...)
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
