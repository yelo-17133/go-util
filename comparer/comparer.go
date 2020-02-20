// ------------------------------------------------------------------------------
// 用于比较两个对象的函数集。
// Golang 不是弱类型语言、不是解释型语言。但弱类型语言、解释型语言也有他们的优点，比如能判断出 1 == "1" 等类型不同、但实际含义相同的表达式。
// 同时，解释型语言可以很方便的在运行时判断两个变量是否相等，比如: let a = 1; let b = "1"; if evel("$a == $b") { ... }
// 这样的特性很灵活、有时候会很高效。
// 此函数集实现类似的功能，可以通过 Compare(a, b interface{}, opr string, option int) 来比较两个对象。函数内部判断他们可能的类型，尝试将他们类型统一后再进行操作符运算。
//
// 该函数的使用场景，参见 jsonValidator 类库。
// ------------------------------------------------------------------------------
package comparer

import (
	"fmt"
	"strings"
	"yelo/go-util/convertor"
)

const (
	Operator_Exist       = "exist"
	Operator_Eq          = "eq"
	Operator_Ueq         = "ueq"
	Operator_Gt          = "gt"
	Operator_Egt         = "egt"
	Operator_Lt          = "lt"
	Operator_Elt         = "elt"
	Operator_In          = "in"
	Operator_NotIn       = "not-in"
	Operator_Contains    = "contains"
	Operator_NotContains = "not-contains"

	Option_None            = iota
	Option_CaseSensitive   = 1
	Option_CaseInsensitive = 2
)

func Compare(a, b interface{}, opr string, options ...int) (bool, error) {
	opr = strings.ToLower(opr)
	basicTypeA, reflectTypeA, reflectValueA := convertor.GetBasicType(a)
	basicTypeB, reflectTypeB, reflectValueB := convertor.GetBasicType(b)

	if opr == Operator_Exist {
		return basicTypeA != convertor.BasicType_Invalid, nil
	}

	if basicTypeA == convertor.BasicType_Unknown {
		return false, fmt.Errorf(`不支持对 %s(%s) 进行运算`, strings.Trim(reflectTypeA.PkgPath()+"."+reflectTypeA.Name(), "."), convertor.ToStringNoError(a))
	} else if basicTypeA == convertor.BasicType_Invalid {
		return false, fmt.Errorf("参数 a 无效")
	}
	if basicTypeA == convertor.BasicType_Unknown {
		return false, fmt.Errorf(`不支持对 %s(%s) 进行运算`, strings.Trim(reflectTypeB.PkgPath()+"."+reflectTypeB.Name(), "."), convertor.ToStringNoError(b))
	} else if basicTypeB == convertor.BasicType_Invalid {
		return false, fmt.Errorf("参数 b 无效")
	}

	switch strings.ToLower(opr) {
	case Operator_In, Operator_NotIn:
		if basicTypeA == convertor.BasicType_String && basicTypeB == convertor.BasicType_String {
			return CompareString(convertor.ToStringNoError(a), convertor.ToStringNoError(b), opr, options...)
		}
		if basicTypeB != convertor.BasicType_Slice {
			return false, fmt.Errorf(`不支持对 %s(%s) 进行 %s 运算`, strings.Trim(reflectTypeB.PkgPath()+"."+reflectTypeB.Name(), "."), convertor.ToStringNoError(b), opr)
		}
		in := false
		for i, n := 0, reflectValueB.Len(); i < n; i++ {
			eq, err := Compare(a, reflectValueB.Index(i).Interface(), Operator_Eq, options...)
			if err != nil {
				return false, err
			} else if eq {
				in = true
				break
			}
		}
		return in == (opr == Operator_In), nil
	case Operator_Contains, Operator_NotContains:
		if basicTypeA == convertor.BasicType_String && basicTypeB == convertor.BasicType_String {
			return CompareString(convertor.ToStringNoError(a), convertor.ToStringNoError(b), opr, options...)
		}
		if basicTypeA != convertor.BasicType_Slice {
			return false, fmt.Errorf(`不支持对 %s(%s) 进行 %s 运算`, strings.Trim(reflectTypeA.PkgPath()+"."+reflectTypeA.Name(), "."), convertor.ToStringNoError(a), opr)
		}
		in := false
		for i, n := 0, reflectValueA.Len(); i < n; i++ {
			eq, err := Compare(reflectValueA.Index(i).Interface(), b, Operator_Eq, options...)
			if err != nil {
				return false, err
			} else if eq {
				in = true
				break
			}
		}
		return in == (opr == Operator_Contains), nil
	case Operator_Eq, Operator_Ueq, Operator_Lt, Operator_Elt, Operator_Gt, Operator_Egt:
		if basicTypeA == convertor.BasicType_Slice {
			return false, fmt.Errorf(`不支持对 %s(%s) 进行 %s 运算`, strings.Trim(reflectTypeA.PkgPath()+"."+reflectTypeA.Name(), "."), convertor.ToStringNoError(a), opr)
		}
		if basicTypeB == convertor.BasicType_Slice {
			return false, fmt.Errorf(`不支持对 %s(%s) 进行 %s 运算`, strings.Trim(reflectTypeB.PkgPath()+"."+reflectTypeB.Name(), "."), convertor.ToStringNoError(b), opr)
		}
		if basicTypeA == convertor.BasicType_Nil {
			return CompareNil(b, opr)
		} else if basicTypeB == convertor.BasicType_Nil {
			return CompareNil(a, opr)
		} else if basicTypeA == basicTypeB {
			return compareSimpleObj(basicTypeA, a, b, opr, options...)
		} else if basicTypeA == convertor.BasicType_Float || basicTypeB == convertor.BasicType_Float {
			return compareSimpleObj(convertor.BasicType_Float, a, b, opr, options...)
		} else if basicTypeA == convertor.BasicType_Uint || basicTypeB == convertor.BasicType_Uint {
			return compareSimpleObj(convertor.BasicType_Uint, a, b, opr, options...)
		} else {
			return compareSimpleObj(convertor.BasicType_Int, a, b, opr, options...)
		}
	default:
		return false, fmt.Errorf("无法识别的运算符: %s", opr)
	}
}

func CompareNil(a interface{}, opr string) (bool, error) {
	n := int64(1)
	if a == nil || a == "" || a == 0 || a == false {
		n = 0
	}
	switch opr {
	case "", Operator_Eq:
		return n == 0, nil
	case Operator_Ueq, Operator_Gt:
		return n != 0, nil
	case Operator_Lt:
		return false, nil
	default:
		return false, fmt.Errorf(`不支持对 nil 进行 %s 运算`, opr)
	}
}

// 比较两个 bool 变量。 opr 指定比较操作符， eq|ueq
func CompareBool(a, b bool, opr string) (bool, error) {
	switch opr {
	case "", Operator_Eq:
		return a == b, nil
	case Operator_Ueq:
		return a != b, nil
	default:
		return false, fmt.Errorf(`不支持对 bool 进行 %s 运算`, opr)
	}
}

// 比较两个 int 变量。 opr 指定比较操作符， eq|ueq|gt|egt|lt|elt
func CompareInt(a, b int64, opr string) (bool, error) {
	switch opr {
	case "", Operator_Eq:
		return a == b, nil
	case Operator_Ueq:
		return a != b, nil
	case Operator_Gt:
		return a > b, nil
	case Operator_Egt:
		return a >= b, nil
	case Operator_Lt:
		return a < b, nil
	case Operator_Elt:
		return a <= b, nil
	default:
		return false, fmt.Errorf(`不支持对 int 进行 %s 运算`, opr)
	}
}

// 比较两个 uint 变量。 opr 指定比较操作符， eq|ueq|gt|egt|lt|elt
func CompareUint(a, b uint64, opr string) (bool, error) {
	switch opr {
	case "", Operator_Eq:
		return a == b, nil
	case Operator_Ueq:
		return a != b, nil
	case Operator_Gt:
		return a > b, nil
	case Operator_Egt:
		return a >= b, nil
	case Operator_Lt:
		return a < b, nil
	case Operator_Elt:
		return a <= b, nil
	default:
		return false, fmt.Errorf(`不支持对 uint 进行 %s 运算`, opr)
	}
}

// 比较两个 float 变量。 opr 指定比较操作符， eq|ueq|gt|egt|lt|elt
func CompareFloat(a, b float64, opr string) (bool, error) {
	switch opr {
	case "", Operator_Eq:
		return a == b, nil
	case Operator_Ueq:
		return a != b, nil
	case Operator_Gt:
		return a > b, nil
	case Operator_Egt:
		return a >= b, nil
	case Operator_Lt:
		return a < b, nil
	case Operator_Elt:
		return a <= b, nil
	default:
		return false, fmt.Errorf(`不支持对 float 进行 %s 运算`, opr)
	}
}

// 比较两个 string 变量。 opr 指定比较操作符， eq|ueq|gt|egt|lt|elt
func CompareString(a, b string, opr string, options ...int) (bool, error) {
	var ignoreCase bool
	for _, n := range options {
		if (n & Option_CaseInsensitive) != 0 {
			ignoreCase = true
		}
	}

	switch opr {
	case "", Operator_Eq:
		if ignoreCase {
			return strings.EqualFold(a, b), nil
		} else {
			return a == b, nil
		}
	case Operator_Ueq:
		if ignoreCase {
			return !strings.EqualFold(a, b), nil
		} else {
			return a != b, nil
		}
	case Operator_Gt:
		if ignoreCase {
			return strings.ToLower(a) > strings.ToLower(b), nil
		} else {
			return a > b, nil
		}
	case Operator_Egt:
		if ignoreCase {
			return strings.ToLower(a) >= strings.ToLower(b), nil
		} else {
			return a >= b, nil
		}
	case Operator_Lt:
		if ignoreCase {
			return strings.ToLower(a) < strings.ToLower(b), nil
		} else {
			return a < b, nil
		}
	case Operator_Elt:
		if ignoreCase {
			return strings.ToLower(a) <= strings.ToLower(b), nil
		} else {
			return a <= b, nil
		}
	case Operator_In:
		if ignoreCase {
			return strings.Index(strings.ToLower(b), strings.ToLower(a)) != -1, nil
		} else {
			return strings.Index(b, a) != -1, nil
		}
	case Operator_NotIn:
		if ignoreCase {
			return strings.Index(strings.ToLower(b), strings.ToLower(a)) == -1, nil
		} else {
			return strings.Index(b, a) == -1, nil
		}
	case Operator_Contains:
		if ignoreCase {
			return strings.Index(strings.ToLower(a), strings.ToLower(b)) != -1, nil
		} else {
			return strings.Index(a, b) != -1, nil
		}
	case Operator_NotContains:
		if ignoreCase {
			return strings.Index(strings.ToLower(a), strings.ToLower(b)) == -1, nil
		} else {
			return strings.Index(a, b) == -1, nil
		}
	default:
		return false, fmt.Errorf(`不支持对 string 进行 %s 运算`, opr)
	}
}

func compareSimpleObj(t convertor.BasicType, a, b interface{}, opr string, options ...int) (bool, error) {
	switch t {
	case convertor.BasicType_Bool:
		valA, err := convertor.ToBool(a)
		if err != nil {
			return false, fmt.Errorf("参数 a 无效: %v", err)
		}
		valB, err := convertor.ToBool(b)
		if err != nil {
			return false, fmt.Errorf("参数 b 无效: %v", err)
		}
		return CompareBool(valA, valB, opr)
	case convertor.BasicType_Int:
		valA, err := convertor.ToInt64(a)
		if err != nil {
			return false, fmt.Errorf("参数 a 无效: %v", err)
		}
		valB, err := convertor.ToInt64(b)
		if err != nil {
			return false, fmt.Errorf("参数 b 无效: %v", err)
		}
		return CompareInt(valA, valB, opr)
	case convertor.BasicType_Uint:
		valA, err := convertor.ToUint64(a)
		if err != nil {
			return false, fmt.Errorf("参数 a 无效: %v", err)
		}
		valB, err := convertor.ToUint64(b)
		if err != nil {
			return false, fmt.Errorf("参数 b 无效: %v", err)
		}
		return CompareUint(valA, valB, opr)
	case convertor.BasicType_Float:
		valA, err := convertor.ToFloat64(a)
		if err != nil {
			return false, fmt.Errorf("参数 a 无效: %v", err)
		}
		valB, err := convertor.ToFloat64(b)
		if err != nil {
			return false, fmt.Errorf("参数 b 无效: %v", err)
		}
		return CompareFloat(valA, valB, opr)
	case convertor.BasicType_String:
		valA := convertor.ToStringNoError(a)
		valB := convertor.ToStringNoError(b)
		return CompareString(valA, valB, opr, options...)
	default:
		return false, fmt.Errorf("无法识别的运算符: %s", opr)
	}
}
