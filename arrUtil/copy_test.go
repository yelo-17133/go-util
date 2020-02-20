package arrUtil

import (
	"sort"
	"testing"
	"yelo/go-util/jsonUtil"
)

func TestCopy(t *testing.T) {
	type abc struct {
		Id   int
		Name string
	}

	arr1 := []*abc{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
		{Id: 3, Name: "c"},
		{Id: 4, Name: "d"},
		{Id: 5, Name: "e"},
	}
	arr2 := make([]*abc, len(arr1))
	copy(arr2, arr1)

	sort.Slice(arr2, func(i, j int) bool {
		return i > j
	})
	for _, item := range arr2 {
		item.Name = item.Name + item.Name + item.Name
	}

	t.Log(jsonUtil.MustMarshalToStringIndent(arr1))
	t.Log(jsonUtil.MustMarshalToStringIndent(arr2))
}
