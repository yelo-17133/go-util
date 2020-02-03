// ------------------------------------------------------------------------------
// 对 jsoniter 的进一步封装
//   1、默认注册弱类型解释器
//   2、简化接口调用
// ------------------------------------------------------------------------------
package jsonUtil

import (
	"github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
	"strconv"
	"strings"
	"unsafe"
)

func init() {
	// 注册弱类型解析，支持将 string("123") 解析为 int(123) 等。
	extra.RegisterFuzzyDecoders()

	// 注册 bool 类型的 fuzzy decoder，支持将 string("true|false") 解析为 bool(true|false)
	jsoniter.RegisterTypeDecoderFunc("bool", func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
		valueType := iter.WhatIsNext()
		var str string
		switch valueType {
		case jsoniter.BoolValue:
			*((*bool)(ptr)) = iter.ReadBool()
		case jsoniter.NumberValue:
			*((*bool)(ptr)) = iter.ReadFloat64() != 0
		case jsoniter.StringValue:
			str = strings.TrimSpace(iter.ReadString())
			if str == "" {
				*((*bool)(ptr)) = false
			} else {
				*((*bool)(ptr)), iter.Error = strconv.ParseBool(strings.Trim(str, "\""))
			}
		default:
			iter.ReportError("fuzzyBoolDecoder", "not bool or string")
		}
	})
}

var (
	defaultApi = NewApi(&jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true,
		ObjectFieldMustBeSimpleString: true,
		SortMapKeys:                   true,
	})

	sortMapKeysApi = NewApi(&jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true,
		ObjectFieldMustBeSimpleString: true,
		SortMapKeys:                   true,
	})
)

func NewApi(config *jsoniter.Config) *Api {
	return &Api{API: config.Froze(), cfg: config}
}

func DefaultApi() *Api {
	return defaultApi
}

func SetDefaultApi(api *Api) {
	defaultApi = api
}

func SortMapKeysApi() *Api {
	return sortMapKeysApi
}

func Unmarshal(data []byte, v interface{}) error {
	return defaultApi.Unmarshal(data, v)
}

func UnmarshalFromString(str string, v interface{}) error {
	return defaultApi.UnmarshalFromString(str, v)
}

func Marshal(v interface{}) ([]byte, error) {
	return defaultApi.Marshal(v)
}

func MustMarshal(v interface{}) []byte {
	return defaultApi.MustMarshal(v)
}

func MarshalIndent(v interface{}, a ...string) ([]byte, error) {
	return defaultApi.MarshalIndent(v, a...)
}

func MustMarshalIndent(v interface{}, a ...string) []byte {
	return defaultApi.MustMarshalIndent(v, a...)
}

func MarshalToString(v interface{}) (string, error) {
	return defaultApi.MarshalToString(v)
}

func MustMarshalToString(v interface{}) string {
	return defaultApi.MustMarshalToString(v)
}

func MarshalToStringIndent(v interface{}, a ...string) (string, error) {
	return defaultApi.MarshalToStringIndent(v, a...)
}

func MustMarshalToStringIndent(v interface{}, a ...string) string {
	return defaultApi.MustMarshalToStringIndent(v, a...)
}

func Get(data []byte, path ...interface{}) jsoniter.Any {
	return defaultApi.Get(data, path...)
}

// ------------------------------------------------------------------------------ Api
type Api struct {
	jsoniter.API
	cfg *jsoniter.Config
}

func (this *Api) Clone() *Api { return NewApi(this.cfg) }

func (this *Api) Unmarshal(data []byte, v interface{}) error {
	if len(data) != 0 {
		return this.API.Unmarshal(data, v)
	}
	return nil
}

func (this *Api) UnmarshalFromString(str string, v interface{}) error {
	if len(str) != 0 {
		return this.API.UnmarshalFromString(str, v)
	}
	return nil
}

func (this *Api) Marshal(v interface{}) ([]byte, error) {
	return this.API.Marshal(v)
}

func (this *Api) MustMarshal(v interface{}) []byte {
	rtn, _ := this.API.Marshal(v)
	return rtn
}

func (this *Api) MarshalIndent(v interface{}, a ...string) ([]byte, error) {
	indent, prefix := "    ", ""
	if n := len(a); n > 0 {
		indent = a[0]
		if n > 1 {
			prefix = a[1]
		}
	}
	rtn, _ := this.API.MarshalIndent(v, prefix, indent)
	return rtn, nil
}

func (this *Api) MustMarshalIndent(v interface{}, a ...string) []byte {
	rtn, _ := this.MarshalIndent(v, a...)
	return rtn
}

func (this *Api) MarshalToString(v interface{}) (string, error) {
	return this.API.MarshalToString(v)
}

func (this *Api) MustMarshalToString(v interface{}) string {
	rtn, _ := this.API.MarshalToString(v)
	return rtn
}

func (this *Api) MarshalToStringIndent(v interface{}, a ...string) (string, error) {
	if val, err := this.MarshalIndent(v, a...); err != nil {
		return "", err
	} else {
		return string(val), nil
	}
}

func (this *Api) MustMarshalToStringIndent(v interface{}, a ...string) string {
	if val, err := this.MarshalIndent(v, a...); err != nil {
		return ""
	} else {
		return string(val)
	}
}

func (this *Api) Get(data []byte, path ...interface{}) jsoniter.Any {
	return this.API.Get(data, path...)
}
