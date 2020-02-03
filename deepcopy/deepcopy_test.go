package deepcopy

import (
	"go-util/jsonUtil"
	"testing"
)

func TestCopyNil(t *testing.T) {
	{
		var a map[string]interface{}
		b, ok := Copy(a).(map[string]interface{})
		if !ok {
			t.Errorf("assert faild: %v", jsonUtil.MustMarshalToString(b))
		}
	}
}
