package convertor

import (
	"fmt"
	"reflect"
)

type BasicType int

const (
	BasicType_Invalid = iota
	BasicType_Nil
	BasicType_Bool
	BasicType_Int
	BasicType_Uint
	BasicType_Float
	BasicType_String
	BasicType_Slice
	BasicType_Map
	BasicType_Struct
	BasicType_Unknown
)

func (this BasicType) String() string {
	switch this {
	case BasicType_Invalid:
		return "invalid"
	case BasicType_Nil:
		return "nil"
	case BasicType_Bool:
		return "bool"
	case BasicType_Int:
		return "int"
	case BasicType_Uint:
		return "uint"
	case BasicType_Float:
		return "float"
	case BasicType_String:
		return "string"
	case BasicType_Slice:
		return "slice"
	case BasicType_Map:
		return "map"
	case BasicType_Struct:
		return "struct"
	case BasicType_Unknown:
		return "others"
	default:
		return fmt.Sprintf("unknown(%d)", int(this))
	}
}

func GetBasicType(a interface{}) (BasicType, reflect.Type, reflect.Value) {
	reflectType := reflect.TypeOf(a)
	reflectValue := reflect.ValueOf(a)
	if a == nil {
		return BasicType_Nil, reflectType, reflectValue
	}

	kind := reflectType.Kind()
	if kind == reflect.Ptr {
		reflectType = reflectType.Elem()
		reflectValue = reflectValue.Elem()
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Invalid:
		return BasicType_Invalid, reflectType, reflectValue
	case reflect.Bool:
		return BasicType_Bool, reflectType, reflectValue
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return BasicType_Int, reflectType, reflectValue
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return BasicType_Uint, reflectType, reflectValue
	case reflect.Float32, reflect.Float64:
		return BasicType_Float, reflectType, reflectValue
	case reflect.String:
		return BasicType_String, reflectType, reflectValue
	case reflect.Slice, reflect.Array:
		return BasicType_Slice, reflectType, reflectValue
	case reflect.Map:
		return BasicType_Map, reflectType, reflectValue
	case reflect.Struct:
		return BasicType_Struct, reflectType, reflectValue
	default:
		return BasicType_Unknown, reflectType, reflectValue
	}
}

func IsEmpty(a interface{}) bool {
	if a == nil {
		return true
	}
	switch t, _, v := GetBasicType(a); t {
	case BasicType_Bool:
		return !v.Bool()
	case BasicType_Int:
		return v.Int() == 0
	case BasicType_Uint:
		return v.Uint() == 0
	case BasicType_Float:
		return v.Float() == 0
	case BasicType_String, BasicType_Slice, BasicType_Map:
		return v.Len() == 0
	}
	return false
}
