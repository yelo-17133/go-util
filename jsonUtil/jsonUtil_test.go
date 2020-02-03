package jsonUtil

import "testing"

func TestUnmarshalFromString(t *testing.T) {
	a := &struct {
		Name string `json:"name"`
	}{}
	str := `{"abc":null}`
	err := UnmarshalFromString(str, a)
	if err != nil {
		t.Error(err)
	} else if a.Name != "" {
		t.Errorf("assert faild: %v", a.Name)
	}
}
