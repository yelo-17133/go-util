package timeUtil

import (
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	GeneralFormat     = "2006-01-02 15:04:05"
	GeneralFormatNano = "2006-01-02 15:04:05.999999999"
)

var (
	Greenwich1970 = time.Unix(0, 0)                                // 格林威治时间 1970-01-01 00:00:00.000000000 (unix = 0)
	Local2000     = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)  // 本地时区时间 2000-01-01 00:00:00.000000000
	EndOf2037     = time.Date(2038, 1, 1, 0, 0, 0, -1, time.Local) // 本地时区时间 2037-12-31 23:59:59.999999999
	Local2000Ms   = ToMs(Local2000)                                // Millisecond of 本地时区时间 2000-01-01 00:00:00.000000000
	EndOf2037Ms   = ToMs(EndOf2037)                                // Millisecond of 本地时区时间 2037-12-31 23:59:59.999999999
	DefaultFormat = GeneralFormat
)

func Max(a time.Time, b ...time.Time) time.Time {
	t := a
	for _, v := range b {
		if t.Before(v) {
			t = v
		}
	}
	return t
}

func Min(a time.Time, b ...time.Time) time.Time {
	t := a
	for _, v := range b {
		if t.After(v) {
			t = v
		}
	}
	return t
}

func ToSecondStr(t time.Time, decimals int) string {
	if decimals == 0 {
		return strconv.FormatInt(t.Unix(), 10)
	} else {
		str := strconv.FormatFloat(float64(t.UnixNano())/float64(time.Second), 'f', decimals, 64)
		if decimals > 0 {
			str = strings.TrimRight(strings.TrimRight(str, "0"), ".")
		}
		return str
	}
}

// 将 time.Time 转换为以秒为单位的浮点数
func ToSecondFloat(t time.Time, decimals int) float64 {
	if decimals == 0 {
		return float64(t.Unix())
	} else {
		v, _ := strconv.ParseFloat(strconv.FormatFloat(float64(t.UnixNano())/float64(time.Second), 'f', decimals, 64), 64)
		return v
	}
}

// 将 time.Time 转换为以毫秒为单位的时间戳整数
func ToMs(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func ToMsStr(t time.Time, decimals int) string {
	if decimals == 0 {
		return strconv.FormatInt(t.UnixNano()/int64(time.Millisecond), 10)
	} else {
		ns := t.UnixNano()
		n, frac := ns/int64(time.Millisecond), ns%int64(time.Millisecond)
		if frac == 0 {
			return strconv.FormatInt(n, 10)
		} else {
			return strconv.FormatInt(n, 10) + "." + strings.TrimRight(strconv.FormatInt(frac, 10), "0")
		}
	}
}

// 将 time.Time 转换为以毫秒为单位的时间戳浮点数
func ToMsFloat(t time.Time, decimals int) float64 {
	if decimals == 0 {
		return float64(t.UnixNano() / int64(time.Millisecond))
	} else {
		ns := t.UnixNano()
		n, frac := ns/int64(time.Millisecond), ns%int64(time.Millisecond)
		v, _ := strconv.ParseFloat(strconv.FormatInt(n, 10)+"."+strconv.FormatInt(frac, 10), 64)
		return v
	}
}

func FromUnixNano(n int64) time.Time {
	return time.Unix(n/int64(time.Second), n%int64(time.Second))
}

// 将以毫秒为单位的时间戳整数转换为 time.Time
func FromMs(n int64) time.Time {
	return time.Unix(n/1000, (n*int64(time.Millisecond))%int64(time.Second))
}

// 将以毫秒为单位的时间戳整数转换为 time.Time
func FromMsFloat(f float64) time.Time {
	return time.Unix(int64(f/1000), int64(f*float64(time.Millisecond))%int64(time.Second))
}

// 将以毫秒为单位的时间戳整数转换为 time.Time
func FromSecondFloat(f float64) time.Time {
	s, ns := math.Modf(f)
	return time.Unix(int64(s), int64(ns*float64(time.Second)))
}

// 获取一天开始的时间（00:00:00）
func BeginOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// 获取一天结束的时间（23:59:59.999999999）
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, -1, t.Location())
}

// 获取一天指定小时数的时间（xx:00:00）
func HourOfDay(t time.Time, hour int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location())
}

// 获取一周的开始时间（周日为一周第一天）
func BeginOfWeek(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()-int(t.Weekday()), 0, 0, 0, 0, t.Location())
}

// 获取一周的结束时间（周六为一周最后一天）
func EndOfWeek(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+7-int(t.Weekday()), 0, 0, 0, -1, t.Location())
}

func BeginOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func EndOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, -1, t.Location())
}

func BeginOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
}

func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January+12, 1, 0, 0, 0, -1, t.Location())
}

func Format(t time.Time, format ...string) string {
	var realFormat string
	if len(format) != 0 {
		realFormat = format[0]
	}
	if realFormat == "" {
		realFormat = DefaultFormat
	}
	return t.Format(realFormat)
}
