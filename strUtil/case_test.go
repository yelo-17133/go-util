package strUtil

import "testing"

func TestSnakeToUpperKebab(t *testing.T) {
	for key, expect := range map[string]string{
		"sso_session_uid": "Sso-Session-Uid",
	} {
		if val := SnakeToUpperKebab(key); val != expect {
			t.Errorf("assert faild: expect %v, but %v", expect, val)
		}
	}
}
