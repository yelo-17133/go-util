package mathUtil

func SumInt64(a []int64) int64 {
	n := int64(0)
	for _, v := range a {
		n += v
	}
	return n
}

func SumInt32(a []int32) int32 {
	n := int32(0)
	for _, v := range a {
		n += v
	}
	return n
}

func SumInt(a []int) int {
	n := 0
	for _, v := range a {
		n += v
	}
	return n
}
