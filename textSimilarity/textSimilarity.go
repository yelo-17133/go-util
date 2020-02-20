package textSimilarity

import (
	"bytes"
	"math"
	"sort"
	"yelo/go-util/arrUtil"
)

type TextSimilarity interface {
	String(a, b string) float64
	Words(a, b []string) float64
	WordsWithWeight(a, b []string, weight map[string]float64) float64
}

func strToWords(a string) []string {
	runes := bytes.Runes([]byte(a))
	n := len(runes)
	w := make([]string, n)
	for i := 0; i < n; i++ {
		w[i] = string(runes[i])
	}
	return w
}

func Cos() TextSimilarity { return &cosSimilarity{} }

type cosSimilarity struct{}

func (this *cosSimilarity) String(a, b string) float64 {
	return this.WordsWithWeight(strToWords(a), strToWords(b), nil)
}

func (this *cosSimilarity) Words(a, b []string) float64 {
	return this.WordsWithWeight(a, b, nil)
}

func (this *cosSimilarity) WordsWithWeight(a, b []string, weight map[string]float64) float64 {
	// 计算所有的 words 并排序
	all := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if arrUtil.IndexOfString(all, v, false) == -1 {
			all = append(all, v)
		}
	}
	for _, v := range b {
		if arrUtil.IndexOfString(all, v, false) == -1 {
			all = append(all, v)
		}
	}
	sort.Strings(all)
	n := len(all)

	// 计算 a、b 的权重向量
	af, bf := make([]float64, n), make([]float64, n)
	for _, v := range a {
		w, ok := weight[v]
		if !ok {
			w = 1
		}
		af[arrUtil.IndexOfString(all, v, false)] += w
	}
	for _, v := range b {
		w, ok := weight[v]
		if !ok {
			w = 1
		}
		bf[arrUtil.IndexOfString(all, v, false)] += w
	}

	// 计算余弦值
	t1, t2, t3 := float64(0), float64(0), float64(0)
	for i := 0; i < n; i++ {
		va, vb := af[i], bf[i]
		t1 += va * vb
		t2 += va * va
		t3 += vb * vb
	}
	if t2 == 0 || t3 == 0 {
		return 0
	} else {
		return t1 / (math.Sqrt(t2) * math.Sqrt(t3))
	}
}
