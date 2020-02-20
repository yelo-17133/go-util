package convertor

import (
	"fmt"
	"testing"
)

func TestGetBasicType(t *testing.T) {
	for _, arr := range [][]interface{}{
		{1, BasicType_Int},
		{[]int{1}, BasicType_Slice},
	} {
		val, expectT := arr[0], BasicType(arr[1].(int))
		gotT, _, _ := GetBasicType(val)
		if gotT != expectT {
			t.Error(fmt.Errorf("assert faild: expect %v, bug %v, val=%v", expectT.String(), gotT.String(), ToStringNoError(val)))
		}
	}
}
