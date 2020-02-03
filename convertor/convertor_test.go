package convertor

import (
	"fmt"
	"math"
	"go-util/jsonUtil"
	"testing"
	"time"
)

// Person has a Name, Age and Address.
type Person struct {
	Name string
	Age  uint
}

func TestToString(t *testing.T) {
	var obj map[string]interface{}
	str := MustToString(obj)
	t.Log(jsonUtil.MustMarshalToString([]interface{}{str}))
}

func TestToBool(t *testing.T) {
	for _, obj := range []interface{}{
		1, "1", -1, "-1", "true", "True", "TRUE", "TrUe", "tRuE", "3", 3, "-3", -3, 1.23, "1.23", -1.23, "-1.23",
	} {
		val, err := ToBool(obj)
		if err != nil {
			t.Error(fmt.Sprintf("error occured: err=%v, obj=%s", err, jsonUtil.MustMarshalToString(obj)))
		} else {
			if val != true {
				t.Error(fmt.Sprintf("accert faild: val=%v, obj=%s", val, jsonUtil.MustMarshalToString(obj)))
			}
		}
	}

	for _, obj := range []interface{}{
		nil, "", 0, "0", "false", "FALSE", "FaLSe", "fAlSe", "-0",
	} {
		val, err := ToBool(obj)
		if err != nil {
			t.Error(fmt.Sprintf("error occured: err=%v, obj=%s", err, jsonUtil.MustMarshalToString(obj)))
		} else {
			if val != false {
				t.Error(fmt.Sprintf("accert faild: val=%v, obj=%s", val, jsonUtil.MustMarshalToString(obj)))
			}
		}
	}

	for _, obj := range []interface{}{
		"-",
		time.Now(),
		[]int{1}, []interface{}{"a"},
		map[string]interface{}{}, map[string]interface{}{"name": "abc"},
		Person{}, &Person{}, Person{Name: "yelo"}, &Person{Name: "yelo"}, Person{Name: "yelo", Age: 24}, &Person{Name: "yelo", Age: 24},
	} {
		val, err := ToBool(obj)
		if err == nil {
			t.Error(fmt.Sprintf("accert faild: val=%v, obj=%s", val, jsonUtil.MustMarshalToString(obj)))
		}
	}
}

func TestToInt(t *testing.T) {
	for obj, expect := range map[interface{}]int64{
		nil:    0,
		true:   1,
		false:  0,
		0:      0,
		1:      1,
		3.2:    3,
		3.8:    3,
		-1:     -1,
		-3.2:   -3,
		-3.8:   -3,
		"":     0,
		"0":    0,
		"1":    1,
		"3.2":  3,
		"3.8":  3,
		"-0":   0,
		"-1":   -1,
		"-3.2": -3,
		"-3.8": -3,
	} {
		val, err := ToInt64(obj)
		if err != nil {
			t.Error(fmt.Sprintf("error occured: err=%v, obj=%s", err, jsonUtil.MustMarshalToString(obj)))
		} else if val != expect {
			t.Error(fmt.Sprintf("accert faild: expect %v, but %v, obj=%s", expect, val, jsonUtil.MustMarshalToString(obj)))
		}
	}

	for _, obj := range []interface{}{
		"-",
		time.Now(),
		[]int{1}, []interface{}{"a"},
		map[string]interface{}{}, map[string]interface{}{"name": "abc"},
		Person{}, &Person{}, Person{Name: "yelo"}, &Person{Name: "yelo"}, Person{Name: "yelo", Age: 24}, &Person{Name: "yelo", Age: 24},
	} {
		val, err := ToInt64(obj)
		if err == nil {
			t.Error(fmt.Sprintf("accert faild: val=%v, item=%s", val, jsonUtil.MustMarshalToString(obj)))
		}
	}
}

func TestToUint(t *testing.T) {
	for obj, expect := range map[interface{}]uint64{
		nil:    0,
		true:   1,
		false:  0,
		"":     0,
		0:      0,
		1:      1,
		3.2:    3,
		3.8:    3,
		-1:     math.MaxUint64,
		-3.2:   math.MaxUint64 - 2,
		-3.8:   math.MaxUint64 - 2,
		"0":    0,
		"1":    1,
		"3.2":  3,
		"3.8":  3,
		"-0":   0,
		"-1":   math.MaxUint64,
		"-3.2": math.MaxUint64 - 2,
		"-3.8": math.MaxUint64 - 2,
	} {
		val, err := ToUint64(obj)
		if err != nil {
			t.Error(fmt.Sprintf("error occured: err=%v, obj=%s", err, jsonUtil.MustMarshalToString(obj)))
		} else if val != expect {
			t.Error(fmt.Sprintf("accert faild: expect %v, but %v, obj=%s", expect, val, jsonUtil.MustMarshalToString(obj)))
		}
	}

	for _, obj := range []interface{}{
		"-",
		time.Now(),
		[]int{1}, []interface{}{"a"},
		map[string]interface{}{}, map[string]interface{}{"name": "abc"},
		Person{}, &Person{}, Person{Name: "yelo"}, &Person{Name: "yelo"}, Person{Name: "yelo", Age: 24}, &Person{Name: "yelo", Age: 24},
	} {
		val, err := ToUint64(obj)
		if err == nil {
			t.Error(fmt.Sprintf("accert faild: val=%v, item=%s", val, jsonUtil.MustMarshalToString(obj)))
		}
	}
}

func TestToFloat(t *testing.T) {
	for obj, expect := range map[interface{}]float64{
		nil:    0,
		true:   1,
		false:  0,
		"":     0,
		0:      0,
		1:      1,
		3.2:    3.2,
		3.8:    3.8,
		-1:     -1,
		-3.2:   -3.2,
		-3.8:   -3.8,
		"0":    0,
		"1":    1,
		"3.2":  3.2,
		"3.8":  3.8,
		"-0":   0,
		"-1":   -1,
		"-3.2": -3.2,
		"-3.8": -3.8,
	} {
		val, err := ToFloat64(obj)
		if err != nil {
			t.Error(fmt.Sprintf("error occured: err=%v, obj=%s", err, jsonUtil.MustMarshalToString(obj)))
		} else if val != expect {
			t.Error(fmt.Sprintf("accert faild: expect %v, but %v, obj=%s", expect, val, jsonUtil.MustMarshalToString(obj)))
		}
	}

	for _, obj := range []interface{}{
		"-",
		time.Now(),
		[]int{1}, []interface{}{"a"},
		map[string]interface{}{}, map[string]interface{}{"name": "abc"},
		Person{}, &Person{}, Person{Name: "yelo"}, &Person{Name: "yelo"}, Person{Name: "yelo", Age: 24}, &Person{Name: "yelo", Age: 24},
	} {
		val, err := ToUint64(obj)
		if err == nil {
			t.Error(fmt.Sprintf("accert faild: val=%v, item=%s", val, jsonUtil.MustMarshalToString(obj)))
		}
	}
}
