package mapUtil

import "testing"

func TestEqual1(t *testing.T) {
	if !Equal(nil, nil) {
		t.Errorf("assert faild")
	}

	if !Equal(nil, nil) {
		t.Errorf("assert faild")
	}

	if !Equal(map[string]string{}, nil) {
		t.Errorf("assert faild")
	}

	func() {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("assert faild")
			}
		}()
		Equal(map[string]string{}, strIntMap{})
	}()

	func() {
		defer func() {
			if e := recover(); e == nil {
				t.Errorf("assert faild")
			}
		}()
		Equal(&strIntMap{}, strIntMap{})
	}()
}

func TestEqual2(t *testing.T) {
	if Equal(strIntMap{"a": 1, "b": 2, "c": 3}, strIntMap{"a": 1, "b": 222}) {
		t.Errorf("assert faild")
	}

	if Equal(&strIntMap{"a": 1, "b": 2, "c": 3}, &strIntMap{"a": 1, "b": 222}) {
		t.Errorf("assert faild")
	}
}

func TestEqual3(t *testing.T) {
	if !Equal(strIntMap{"a": 1, "b": 2, "c": 3}, strIntMap{"a": 1, "b": 2, "c": 3}) {
		t.Errorf("assert faild")
	}

	if !Equal(&strIntMap{"a": 1, "b": 2, "c": 3}, &strIntMap{"a": 1, "b": 2, "c": 3}) {
		t.Errorf("assert faild")
	}
}

func TestEqual4(t *testing.T) {
	if Equal(strIntMap{"a": 1, "b": 2, "c": 0}, strIntMap{"a": 1, "b": 2}) {
		t.Errorf("assert faild")
	}
}
