package jsonValidator

import (
	"fmt"
	"go-util/convertor"
	"go-util/jsonUtil"
	"testing"
)

func TestValidator_GetValue(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	obj, type_, err := validator.GetValue("data.id")
	if err != nil {
		t.Error(fmt.Errorf("error occured: %v", err))
		return
	}

	if type_ != convertor.BasicType_Slice {
		t.Error("assert faild")
	}
	if len(obj.([]interface{})) != 3 {
		t.Error("assert faild")
	}
}

func TestValidator_GetValue_1(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	arr := make([]interface{}, 0)
	_, _, err := validator.doGetValue(validator.jsonObj, "data.items.*.id", &arr, "")
	if err != nil {
		t.Error(fmt.Errorf("error occured: %v", err))
		return
	}
	t.Log(jsonUtil.MustMarshalToStringIndent(arr))
}

func TestValidator_GetValue_2(t *testing.T) {
	validator, ok := initValidator(t, `
[
	{"id": 1, "name": "a", "data": {"str": "aa"}},
	{"id": 2, "name": "b", "data": {"str": "bb"}},
	{"id": 3, "name": "c", "data": {"str": "cc"}},
	{"id": 4, "name": "d", "data": {"str": "dd"}},
	{"id": 5, "name": "e", "data": {"str": "ee"}}
]
`)
	if !ok {
		return
	}

	arr := make([]interface{}, 0)
	_, _, err := validator.doGetValue(validator.jsonObj, "*.data.str", &arr, "")
	if err != nil {
		t.Error(fmt.Errorf("error occured: %v", err))
		return
	}
	t.Log(jsonUtil.MustMarshalToStringIndent(arr))
}

func TestValidator_Validate_1(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"code", "eq", 0},
		{"code", "eq", ""},
		{"code", "eq", "0"},
		{"code", "ueq", 0, false},
		{"code", "egt", 0},
		{"code", "elt", 0},
		{"code", "gt", 0, false},
		{"code", "lt", 0, false},
		{"code", "in", []int{0, 304}},
		{"code", "not-in", []int{-1, 304}},
		{"code", "in", []int{-1, 304}, false},
		{"code", "not-in", []int{0, 304}, false},
	} {
		path, opr, val, expectOk := arr[0].(string), arr[1].(string), arr[2], true
		if len(arr) > 3 {
			expectOk = arr[3].(bool)
		}

		ok, err := validator.Validate(path, opr, val, 0)
		if err != nil {
			t.Error(fmt.Errorf("error occured: %v，path=%v, opr=%v, val=%v", err, path, opr, convertor.MustToString(val)))
		} else if ok != expectOk {
			t.Error(fmt.Errorf("assert faild: expect %v, but %v，path=%v, opr=%v, val=%v", expectOk, ok, path, opr, convertor.MustToString(val)))
		}
	}
}

func TestValidator_Validate_2(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"data.userId", "ueq", 0},
		{"data.userId", "gt", 0},
		{"data.clientType", "in", []int{1, 2, 4, 3, 5}},
	} {
		path, opr, val, expectOk := arr[0].(string), arr[1].(string), arr[2], true
		if len(arr) > 3 {
			expectOk = arr[3].(bool)
		}

		ok, err := validator.Validate(path, opr, val, 0)
		if err != nil {
			t.Error(fmt.Errorf("error occured: %v，path=%v, opr=%v, val=%v", err, path, opr, convertor.MustToString(val)))
		} else if ok != expectOk {
			t.Error(fmt.Errorf("assert faild: expect %v, but %v，path=%v, opr=%v, val=%v", expectOk, ok, path, opr, convertor.MustToString(val)))
		}
	}
}

func TestValidator_Validate_3(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"data.id.0", "exist", nil},
		{"data.id.3", "exist", nil, false},
	} {
		path, opr, val, expectOk := arr[0].(string), arr[1].(string), arr[2], true
		if len(arr) > 3 {
			expectOk = arr[3].(bool)
		}

		ok, _ := validator.Validate(path, opr, val, 0)
		if ok != expectOk {
			t.Error(fmt.Errorf("assert faild: expect %v, but %v，path=%v, opr=%v, val=%v", expectOk, ok, path, opr, convertor.MustToString(val)))
		}
	}
}

func TestValidator_Validate_4(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"data.id.0", "eq", 1},
		{"data.id.1", "eq", 2},
	} {
		path, opr, val, expectOk := arr[0].(string), arr[1].(string), arr[2], true
		if len(arr) > 3 {
			expectOk = arr[3].(bool)
		}

		ok, err := validator.Validate(path, opr, val, 0)
		if err != nil {
			t.Error(fmt.Errorf("error occured: %v，path=%v, opr=%v, val=%v", err, path, opr, convertor.MustToString(val)))
		} else if ok != expectOk {
			t.Error(fmt.Errorf("assert faild: expect %v, but %v，path=%v, opr=%v, val=%v", expectOk, ok, path, opr, convertor.MustToString(val)))
		}
	}
}

func TestValidator_Validate_5(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"data.id", "contains", 1},
		{"data.id", "contains", 2},
		{"data.id", "contains", 3},
	} {
		path, opr, val, expectOk := arr[0].(string), arr[1].(string), arr[2], true
		if len(arr) > 3 {
			expectOk = arr[3].(bool)
		}

		ok, err := validator.Validate(path, opr, val, 0)
		if err != nil {
			t.Error(fmt.Errorf("error occured: %v，path=%v, opr=%v, val=%v", err, path, opr, convertor.MustToString(val)))
		} else if ok != expectOk {
			t.Error(fmt.Errorf("assert faild: expect %v, but %v，path=%v, opr=%v, val=%v", expectOk, ok, path, opr, convertor.MustToString(val)))
		}
	}
}

func TestValidator_Validate_Error(t *testing.T) {
	validator, ok := initValidator(t)
	if !ok {
		return
	}

	for _, arr := range [][]interface{}{
		{"code", "eq", []int{0}},
		{"code", "asfdasdf", 0},
		{"code", "contains", 0},
		{"code", "contains", []int{0, 304}},
		{"code", "not-in", 0},
	} {
		path, opr, val := arr[0].(string), arr[1].(string), arr[2]

		ok, err := validator.Validate(path, opr, val, 0)
		if err == nil {
			t.Error(fmt.Errorf("should have error: path=%v, opr=%v, val=%v, ok=%v", path, opr, convertor.MustToString(val), ok))
		}
	}
}

func TestValidator_GetMapValue(t *testing.T) {
	a, _ := New(map[string]interface{}{
		"device": map[string]interface{}{
			"brand":  "cisco",
			"module": "WS-C3750X-48U-E",
			"abc":    nil,
		},
	})

	if v, _, err := a.GetValue("device"); err != nil {
		t.Error(err)
	} else {
		t.Logf("device: %v", jsonUtil.MustMarshalToString(v))
	}

	if v, _, err := a.GetValue("device.brand"); err != nil {
		t.Error(err)
	} else {
		t.Logf("device.brand: %v", jsonUtil.MustMarshalToString(v))
	}

	if v, _, err := a.GetValue("device.abc"); err != nil {
		t.Error(err)
	} else {
		t.Logf("device.abc: %v", jsonUtil.MustMarshalToString(v))
	}
}

func initValidator(t *testing.T, jsonStr ...string) (*validatorImpl, bool) {
	var str string
	if len(jsonStr) != 0 {
		str = jsonStr[0]
	}
	if str == "" {
		str = `
{
    "code": 0,
    "message": "",
    "data": {
        "userId": 123,
        "clientType": 1,
        "deviceId": 123,
        "versionModule": "voice",
        "versionCode": 123,
        "id": [
            1,
            2,
            3
        ],
        "items": [
            {
                "id": 1,
                "name": "a"
            },
            {
                "id": 2,
                "name": "b"
            },
            {
                "id": 3,
                "name": "c"
            },
            {
                "id": 4,
                "name": "d"
            },
            {
                "id": 5,
                "name": "e"
            }
        ]
    }
}
`
	}
	val, err := New(str)
	if err != nil {
		t.Error(fmt.Errorf("initValidator error: %v", err))
	}
	return val.(*validatorImpl), err == nil
}
