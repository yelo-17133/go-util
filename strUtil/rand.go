package strUtil

import (
	"math/rand"
	"sync/atomic"
	"time"
)

var seed = time.Now().UnixNano()

func Rand(length int) string {
	return RandB(length, "0123456789abcdefghijklmnopqrstuvwxyz")
}

func RandB(length int, bucket string) string {
	bytes := []byte(bucket)
	size := len(bytes)
	result := make([]byte, length)
	atomic.AddInt64(&seed, 1)
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < length; i++ {
		result[i] = bytes[r.Intn(size)]
	}
	return string(result)
}
