// ------------------------------------------------------------------------------
// Json 表达式验证器。
// 主要目的：在某些情况下，允许 Golang 像弱类型语言、解释型语言那样，在不重新编译代码的情况下，仅通过修改配置数据中的 “Json 验证表达式”，即可灵活的实现逻辑控制。
// 举例：
//   C2 接入层中，保存了每个用户的 Session 会话数据，里面包含用户的信息、以及客户端的模块名称和版本号。
//   只要通过灵活的配置规则，实现类似 “将一部分用户的请求发送到另一个独立的 Api 代理” 以实现功能灰度。
//   而且，规则还可以灵活配置，无论如何修灰度规则，都不需要接入层重新编译发版。可最大限度降低接入层发版频率。
// ------------------------------------------------------------------------------
package jsonValidator

import (
	"fmt"
	"github.com/json-iterator/go"
	"math"
	"strconv"
	"strings"
	"yelo/go-util/comparer"
	"yelo/go-util/convertor"
	"yelo/go-util/jsonUtil"
)

type Validator interface {
	// 获取要验证的 Json 对象
	JsonObj() jsoniter.Any
	// 获取指定路径对应的值。
	//   当路径不存在时返回错误。
	GetValue(path string) (interface{}, convertor.BasicType, error)
	// 验证指定路径对应的值是否符合表达式。
	//   若指定路径存在、且能够与 val 进行对应的比较操作，则返回比较结果；
	//   若路径不存在、路径对应的值与 val 之间无法进行对应的比较（比如 int 之间无法进行 contains 比较等），则返回错误。
	Validate(path string, operator string, val interface{}, option int) (bool, error)
}

type validatorImpl struct {
	obj     interface{}
	jsonObj jsoniter.Any
}

func New(obj interface{}) (Validator, error) {
	this := &validatorImpl{obj: obj}
	if obj == nil {
		this.jsonObj = jsoniter.Wrap(nil)
	} else {
		switch t := obj.(type) {
		case []byte:
			this.jsonObj = jsoniter.Get(t)
		case string:
			this.jsonObj = jsoniter.Get([]byte(t))
		default:
			this.jsonObj = jsoniter.Get(jsonUtil.MustMarshal(obj))
		}
	}
	if this.jsonObj.ValueType() == jsoniter.InvalidValue {
		return nil, fmt.Errorf("不是有效的 Json 格式")
	}
	return this, nil
}

func (this *validatorImpl) JsonObj() jsoniter.Any {
	return this.jsonObj
}

func (this *validatorImpl) Validate(path string, operator string, val interface{}, option int) (bool, error) {
	if pathVal, _, err := this.GetValue(path); err != nil {
		return false, err
	} else {
		return comparer.Compare(pathVal, val, operator, option)
	}
}

func (this *validatorImpl) GetValue(path string) (interface{}, convertor.BasicType, error) {
	arr := make([]interface{}, 0)
	isArray, basicType, err := this.doGetValue(this.jsonObj, path, &arr, "")
	if isArray {
		return arr, convertor.BasicType_Slice, err
	} else if len(arr) != 0 {
		return arr[0], basicType, err
	} else {
		return nil, basicType, err
	}
}

func (this *validatorImpl) doGetValue(jsonObj jsoniter.Any, path string, arr *[]interface{}, jsonObjPath string) (isArray bool, basicType convertor.BasicType, err error) {
	pos := strings.Index(path, "*")
	if pos == -1 {
		_, v, b, err := this.getOneValue(jsonObj, path)
		if err != nil {
			return false, b, err
		} else {
			*arr = append(*arr, v)
			return false, b, nil
		}
	}

	prefix, suffix, arrPath := strings.TrimRight(path[:pos], "."), strings.TrimLeft(path[pos+1:], "."), ""
	var arrObj jsoniter.Any
	if prefix == "" {
		arrObj = jsonObj
		arrPath = jsonObjPath
	} else {
		arrObj = jsonObj.Get(this.splitPath(prefix)...)
		arrPath = strings.TrimLeft(jsonObjPath+"."+prefix, ".")
	}
	if valType := arrObj.ValueType(); valType == jsoniter.InvalidValue {
		return false, convertor.BasicType_Invalid, fmt.Errorf("路径 '%v' 无效", arrPath)
	} else if valType != jsoniter.ArrayValue {
		return false, convertor.BasicType_Invalid, fmt.Errorf("路径 '%v' 不是数组格式", arrPath)
	}

	for i, n := 0, arrObj.Size(); i < n; i++ {
		arrItemObj := arrObj.Get(i)
		if suffix == "" {
			*arr = append(*arr, arrItemObj)
		} else {
			if _, b, err := this.doGetValue(arrItemObj, suffix, arr, arrPath+strconv.Itoa(i)); err != nil {
				return true, b, err
			}
		}
	}

	return true, convertor.BasicType_Slice, nil
}

func (this *validatorImpl) getOneValue(jsonObj jsoniter.Any, path string) (jsoniter.Any, interface{}, convertor.BasicType, error) {
	pathSegs := make([]interface{}, 0, 16)
	for _, str := range strings.Split(path, ".") {
		if n, e := strconv.ParseInt(str, 10, 64); e == nil {
			pathSegs = append(pathSegs, int(n))
		} else {
			pathSegs = append(pathSegs, str)
		}
	}

	obj := jsonObj.Get(pathSegs...)
	switch obj.ValueType() {
	case jsoniter.InvalidValue:
		return obj, nil, convertor.BasicType_Invalid, fmt.Errorf("路径 %s 无效", path)
	case jsoniter.StringValue:
		return obj, obj.ToString(), convertor.BasicType_String, nil
	case jsoniter.NumberValue:
		f := obj.ToFloat64()
		n, frac := math.Modf(f)
		if math.Abs(frac) < 0.000001 {
			return obj, int64(n), convertor.BasicType_Int, nil
		} else {
			return obj, f, convertor.BasicType_Float, nil
		}
	case jsoniter.NilValue:
		return obj, nil, convertor.BasicType_Nil, nil
	case jsoniter.BoolValue:
		return obj, obj.ToBool(), convertor.BasicType_Bool, nil
	case jsoniter.ArrayValue:
		arr := make([]interface{}, 0)
		obj.ToVal(&arr)
		return obj, arr, convertor.BasicType_Slice, nil
	case jsoniter.ObjectValue:
		dict := make(map[string]interface{})
		obj.ToVal(&dict)
		return obj, dict, convertor.BasicType_Map, nil
	default:
		return obj, nil, convertor.BasicType_Unknown, fmt.Errorf("路径 %s 无效", path)
	}
}

func (this *validatorImpl) splitPath(path string) []interface{} {
	pathSegs := make([]interface{}, 0, 16)
	for _, str := range strings.Split(path, ".") {
		if n, e := strconv.ParseInt(str, 10, 64); e == nil {
			pathSegs = append(pathSegs, int(n))
		} else {
			pathSegs = append(pathSegs, str)
		}
	}
	return pathSegs
}
