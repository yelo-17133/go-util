package textSimilarity

import (
	"fmt"
	"strings"
	"testing"
)

func TestCosString(t *testing.T) {
	s := Cos().String("芝麻", "芝麻。")
	t.Log(s)

	s = Cos().String("芝麻开门", "芝麻开门。")
	t.Log(s)

	s = Cos().String("芝麻开门芝麻开门", "芝麻开门芝麻开门。")
	t.Log(s)

	s = Cos().String("芝麻", "值麻")
	t.Log(s)

	s = Cos().String("芝麻开门", "值麻开门")
	t.Log(s)
}

func TestCosString2(t *testing.T) {
	for _, str := range []string{"调试模式", "调试模式。", "调试后模式。", "调试模式了", "开启调试模式"} {
		s := Cos().String("调试模式", str)
		t.Log(s)
	}
}

func TestStrToWords(t *testing.T) {
	for _, str := range []string{"调试模式", "调试模式。", "调试后模式。", "调试模式了"} {
		words := strToWords(str)
		if tmp := strings.Join(words, ""); tmp != str {
			t.Error(fmt.Sprintf("assert faild: expect %v, but %v", str, tmp))
		}
	}
}
