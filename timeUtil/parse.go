package timeUtil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var knownFormat = map[string]time.Duration{
	"year":        0,
	"month":       0,
	"day":         24 * time.Hour,
	"d":           24 * time.Hour,
	"hour":        time.Hour,
	"h":           time.Hour,
	"minute":      time.Minute,
	"min":         time.Minute,
	"second":      time.Second,
	"sec":         time.Second,
	"millisecond": time.Millisecond,
	"ms":          time.Millisecond,
	"microsecond": time.Microsecond,
	"us":          time.Microsecond,
	"nanosecond":  time.Nanosecond,
	"ns":          time.Nanosecond,
}

// 将字符串转换为 time.Time 类型。支持的格式如下：
//   数字:
//      如果小于 MaxInt32，则以秒为单位转换； 否则以毫秒为单位转换。
//   预定义格式:
//      now|today|yesterday|tommorow: 当前时间、当天开始时间、昨天开始时间、明天开始时间
//      {n}{unit}: 当前时间 增加 {n} 个 {unit} 后的结果。 n 可以为负数、 unit 支持 year|month|day(d)|hour(h)|minute(min)|second(sec)|millisecond(ms)|microsecond(us)|nanosecond(ns)
//      以上格式可以通过英文逗号串联，比如 “tommorow,2hour,-3min” 表示 “明天01:57”
//   [yyyy-]MM-dd 格式
//   [yyyy-]MM-dd HH:mm[:ss] 格式
func Parse(str string) (time.Time, error) {
	str = strings.ToLower(strings.TrimSpace(str))

	// 如果是数字，则按照秒或者毫秒处理
	if n, err := strconv.ParseInt(str, 10, 64); err == nil {
		if n >= math.MaxInt32 {
			return FromMs(n), nil
		} else {
			return time.Unix(n, 0), nil
		}
	}

	// 处理 常量字符串 以及 (+-)(N)(Unit) 格式的字符串，例如 5day、-2hour
	if t, ok, err := tryParseKnownFormat(str); err != nil || ok {
		return t, err
	}

	// 处理 Y-m-d h:i:s 格式
	var dateStr, timeStr, msStr string
	if pos := strings.Index(str, " "); pos != -1 {
		dateStr = str[:pos]
		ss := str[pos+1:]
		if pos = strings.Index(ss, "."); pos != -1 {
			timeStr = ss[:pos]
			msStr = ss[pos+1:]
		} else {
			timeStr = ss
		}
	} else {
		dateStr = str
	}

	var yearStr, monthStr, dayStr, hourStr, minStr, secStr, nsecStr string
	tmp := strings.Split(dateStr, "-")
	switch len(tmp) {
	case 3:
		yearStr, monthStr, dayStr = tmp[0], tmp[1], tmp[2]
	case 2:
		monthStr, dayStr = tmp[0], tmp[1]
	}
	if timeStr != "" {
		tmp = strings.Split(timeStr, ":")
		switch len(tmp) {
		case 3:
			hourStr, minStr, secStr = tmp[0], tmp[1], tmp[2]
			if msStr != "" {
				n := len(msStr)
				if n == 9 {
					nsecStr = msStr
				} else if n > 9 {
					nsecStr = msStr[:9]
				} else {
					nsecStr = msStr + strings.Repeat("0", 9-n)
				}
			}
		case 2:
			hourStr, minStr = tmp[0], tmp[1]
		}
	}

	formatError := fmt.Errorf("格式不正确: %v", str)
	var year, month, day, hour, min, sec, nsec int
	var err error
	if yearStr != "" {
		if year, err = strconv.Atoi(yearStr); err != nil {
			return time.Time{}, formatError
		}
	}
	if monthStr == "" || dayStr == "" {
		return time.Time{}, formatError
	} else {
		if month, err = strconv.Atoi(monthStr); err != nil {
			return time.Time{}, formatError
		}
		if day, err = strconv.Atoi(dayStr); err != nil {
			return time.Time{}, formatError
		}
	}
	if hourStr != "" || minStr != "" || secStr != "" {
		if hour, err = strconv.Atoi(hourStr); err != nil {
			return time.Time{}, formatError
		}
		if min, err = strconv.Atoi(minStr); err != nil {
			return time.Time{}, formatError
		}
		if secStr != "" {
			if sec, err = strconv.Atoi(secStr); err != nil {
				return time.Time{}, formatError
			}
			if nsecStr != "" {
				if nsec, err = strconv.Atoi(nsecStr); err != nil {
					return time.Time{}, formatError
				}
			}
		}
	}
	if year == 0 {
		year = time.Now().Year()
	}

	return time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local), nil
}

func tryParseKnownFormat(s string) (time.Time, bool, error) {
	t := time.Now()
	for _, str := range strings.Split(s, ",") {
		if str = strings.TrimSpace(str); str == "" {
			continue
		}
		switch str {
		case "now":
			t = time.Now()
		case "today":
			t = BeginOfDay(time.Now())
		case "yesterday":
			t = BeginOfDay(time.Now()).Add(-24 * time.Hour)
		case "tommorow":
			t = BeginOfDay(time.Now()).Add(24 * time.Hour)
		default:
			matched := false
			for name, unit := range knownFormat {
				if strings.HasSuffix(str, name) {
					matched = true
					nStr := str[:len(str)-len(name)]
					if n, err := strconv.ParseInt(strings.TrimSpace(nStr), 10, 64); err != nil {
						return time.Time{}, false, fmt.Errorf("无法将 %v 转换为数字", nStr)
					} else if n != 0 {
						if unit != 0 {
							t = t.Add(time.Duration(n) * unit)
						} else if name == "year" {
							t = time.Date(t.Year()+int(n), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
						} else if name == "month" {
							t = time.Date(t.Year(), t.Month()+time.Month(n), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
						}
					}
					break
				}
			}
			if !matched {
				return time.Time{}, false, nil
			}
		}
	}
	return t, true, nil
}
