package mathUtil

import (
	"math"
	"testing"
)

func TestRound(t *testing.T) {
	for _, v := range []float64{
		-0.7,
		-0.5,
		-0.2,
		0,
		0.2,
		0.5,
		0.7,
	} {
		n := math.Round(v)
		t.Logf("%v: %v", v, n)
	}
}
