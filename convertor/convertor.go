// ------------------------------------------------------------------------------
// 类型转换函数集
// ------------------------------------------------------------------------------
package convertor

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"yelo/go-util/jsonUtil"
	"yelo/go-util/timeUtil"
)

func ToBool(a interface{}) (bool, error) {
	if a == nil {
		return false, nil
	}

	t, reflectType, reflectValue := GetBasicType(a)
	switch t {
	case BasicType_Bool:
		return reflectValue.Bool(), nil
	case BasicType_Int:
		return reflectValue.Int() != 0, nil
	case BasicType_Uint:
		return reflectValue.Uint() != 0, nil
	case BasicType_Float:
		return reflectValue.Float() != 0, nil
	case BasicType_String:
		str := a.(string)
		if str == "" {
			return false, nil
		} else if v, err := strconv.ParseBool(strings.ToLower(str)); err == nil {
			return v, nil
		} else if v, err := strconv.ParseFloat(str, 64); err == nil {
			return v != 0, nil
		}
		return false, fmt.Errorf("can't convert string(%s) to bool", str)
	default:
		return false, fmt.Errorf("can't convert %s(%v) to bool", strings.Trim(reflectType.PkgPath()+"."+reflectType.Name(), "."), MustToString(a))
	}
}

func ToInt64(a interface{}) (int64, error) {
	if a == nil {
		return 0, nil
	}

	t, reflectType, reflectValue := GetBasicType(a)
	switch t {
	case BasicType_Bool:
		if reflectValue.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case BasicType_Int:
		return reflectValue.Int(), nil
	case BasicType_Uint:
		return int64(reflectValue.Uint()), nil
	case BasicType_Float:
		return int64(reflectValue.Float()), nil
	case BasicType_String:
		str := a.(string)
		if str == "" {
			return 0, nil
		} else if v, err := strconv.ParseInt(str, 10, 64); err == nil {
			return v, nil
		} else if v, err := strconv.ParseFloat(str, 64); err == nil {
			return int64(v), nil
		} else if v, err := strconv.ParseBool(strings.ToLower(str)); err == nil {
			if v {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("can't convert string(%s) to int", str)
	default:
		return 0, fmt.Errorf("can't convert %s(%v) to int", strings.Trim(reflectType.PkgPath()+"."+reflectType.Name(), "."), MustToString(a))
	}
}

func ToInt32(a interface{}) (int32, error) {
	v, err := ToInt64(a)
	return int32(v), err
}

func ToInt(a interface{}) (int, error) {
	v, err := ToInt64(a)
	return int(v), err
}

func ToUint64(a interface{}) (uint64, error) {
	if a == nil {
		return 0, nil
	}

	t, reflectType, reflectValue := GetBasicType(a)
	switch t {
	case BasicType_Bool:
		if reflectValue.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case BasicType_Int:
		return uint64(reflectValue.Int()), nil
	case BasicType_Uint:
		return reflectValue.Uint(), nil
	case BasicType_Float:
		return uint64(reflectValue.Float()), nil
	case BasicType_String:
		str := a.(string)
		if str == "" {
			return 0, nil
		} else if v, err := strconv.ParseUint(str, 10, 64); err == nil {
			return v, nil
		} else if v, err := strconv.ParseFloat(str, 64); err == nil {
			return uint64(v), nil
		} else if v, err := strconv.ParseBool(strings.ToLower(str)); err == nil {
			if v {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("can't convert string(%s) to uint", str)
	default:
		return 0, fmt.Errorf("can't convert %s(%v) to uint", strings.Trim(reflectType.PkgPath()+"."+reflectType.Name(), "."), MustToString(a))
	}
}

func ToUint32(a interface{}) (uint32, error) {
	v, err := ToUint64(a)
	return uint32(v), err
}

func ToUint(a interface{}) (uint, error) {
	v, err := ToUint64(a)
	return uint(v), err
}

func ToFloat64(a interface{}) (float64, error) {
	if a == nil {
		return 0, nil
	}

	t, reflectType, reflectValue := GetBasicType(a)
	switch t {
	case BasicType_Bool:
		if reflectValue.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case BasicType_Int:
		return float64(reflectValue.Int()), nil
	case BasicType_Uint:
		return float64(reflectValue.Uint()), nil
	case BasicType_Float:
		return reflectValue.Float(), nil
	case BasicType_String:
		str := a.(string)
		if str == "" {
			return 0, nil
		} else if v, err := strconv.ParseFloat(str, 64); err == nil {
			return v, nil
		} else if v, err := strconv.ParseBool(strings.ToLower(str)); err == nil {
			if v {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("can't convert string(%s) to float", str)
	default:
		return 0, fmt.Errorf("can't convert %s(%v) to float", strings.Trim(reflectType.PkgPath()+"."+reflectType.Name(), "."), MustToString(a))
	}
}

func ToFloat32(a interface{}) (float32, error) {
	v, err := ToFloat64(a)
	return float32(v), err
}

func ToString(a interface{}) (string, error) {
	if a == nil {
		return "", nil
	}

	switch t := a.(type) {
	case []byte:
		return string(t), nil
	case string:
		return t, nil
	case bool:
		return strconv.FormatBool(t), nil
	case time.Time:
		return t.Format("2006-01-02 15:04:05"), nil
	case timeUtil.JsonTime:
		return t.Format("2006-01-02 15:04:05"), nil
	}

	switch t, _, reflectValue := GetBasicType(a); t {
	case BasicType_Bool:
		return strconv.FormatBool(reflectValue.Bool()), nil
	case BasicType_Int:
		return strconv.FormatInt(reflectValue.Int(), 10), nil
	case BasicType_Uint:
		return strconv.FormatUint(reflectValue.Uint(), 10), nil
	case BasicType_Float:
		return strconv.FormatFloat(reflectValue.Float(), 'f', -1, 64), nil
	case BasicType_String:
		return reflectValue.String(), nil
	default:
		str, err := jsonUtil.SortMapKeysApi().MarshalToString(a)
		if err != nil {
			return "", err
		} else if lowerStr := strings.ToLower(str); lowerStr == "nil" || lowerStr == "null" || str == "[]" || str == "{}" {
			str = ""
		}
		return str, nil
	}
}

func MustToString(a interface{}) string {
	v, _ := ToString(a)
	return v
}

func MustToBool(a interface{}) bool {
	v, _ := ToBool(a)
	return v
}

func MustToInt64(a interface{}) int64 {
	v, _ := ToInt64(a)
	return v
}

func MustToInt32(a interface{}) int32 {
	v, _ := ToInt64(a)
	return int32(v)
}

func MustToInt(a interface{}) int {
	v, _ := ToInt64(a)
	return int(v)
}

func MustToUint64(a interface{}) uint64 {
	v, _ := ToUint64(a)
	return v
}

func MustToUint32(a interface{}) uint32 {
	v, _ := ToUint64(a)
	return uint32(v)
}

func MustToUint(a interface{}) uint {
	v, _ := ToUint64(a)
	return uint(v)
}

func MustToFloat64(a interface{}) float64 {
	v, _ := ToFloat64(a)
	return v
}

func MustToFloat32(a interface{}) float32 {
	v, _ := ToFloat64(a)
	return float32(v)
}
