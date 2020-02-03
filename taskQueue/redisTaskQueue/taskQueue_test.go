package redisTaskQueue

import (
	"fmt"
	"github.com/go-redis/redis"
	"go-util/_utilTest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_utilTest.Init()
	m.Run()
}

func TestQueue(t *testing.T) {
	topic, queue := "test-ok", NewHandler(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), nil)
	added, handled := int32(0), int32(0)
	var start time.Time
	var err error

	start = time.Now()
	for i := 0; i < 1000; i++ {
		queue.Add(topic, strconv.Itoa(i))
		atomic.AddInt32(&added, 1)
	}
	t.Log(fmt.Sprintf("add %v, timespan=%v", added, time.Now().Sub(start)))

	err = queue.RegisterHandler(topic, func(topic string, data string) (success bool) {
		atomic.AddInt32(&handled, 1)
		return true
	}, &HandlerOptions{Worker: 111})
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}

	start = time.Now()
	if err = queue.Start(); err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	for {
		n, _ := queue.Count(topic)
		if n == 0 {
			break
		} else {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("topic: %v\n", n)
		}
	}
	t.Log(fmt.Sprintf("handled=%v, timespan=%v", handled, time.Now().Sub(start)))

	queue.Stop()
}
