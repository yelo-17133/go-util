package redisDelayTaskQueue

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

func TestQueueImpl_Add(t *testing.T) {
	adding, count := 0, 500
	tasks := make([]*Task, 0, count)
	doTestQueueImplAdd(t, "test-ok", count, func(queue QueueHandler, topic string, i int) {
		queue.Add(topic, strconv.Itoa(i+adding), 0)
	}, func(topic string, task *Task) (finished bool, retryAfter time.Duration, err error) {
		tasks = append(tasks, task)
		return true, 0, nil
	})
	if n := len(tasks); n != count {
		t.Error(fmt.Errorf("assert faild: expect %v, but %v", count, n))
	}
	for i := 0; i < count; i++ {
		s := strconv.Itoa(i + adding)
		var val *Task
		for _, item := range tasks {
			if item.Data == s {
				val = item
				break
			}
		}
		if val == nil {
			t.Error(fmt.Errorf("task data %v not found", s))
		}
	}
}

func TestQueueImpl_AddWithKey(t *testing.T) {
	adding, count := 10000, 500
	tasks := make([]*Task, 0, count)
	doTestQueueImplAdd(t, "test-ok", count, func(queue QueueHandler, topic string, i int) {
		queue.AddWithKey(topic, strconv.Itoa(i+adding), "", 0)
	}, func(topic string, task *Task) (finished bool, retryAfter time.Duration, err error) {
		tasks = append(tasks, task)
		return true, 0, nil
	})
	if n := len(tasks); n != count {
		t.Error(fmt.Errorf("assert faild: expect %v, but %v", count, n))
	}
	for i := 0; i < count; i++ {
		s := strconv.Itoa(i + adding)
		var val *Task
		for _, item := range tasks {
			if item.Key == s && item.Data == "" {
				val = item
				break
			}
		}
		if val == nil {
			t.Error(fmt.Errorf("task data %v not found", s))
		}
	}
}

func TestQueueImpl_AddWithKeyAndData(t *testing.T) {
	adding, count := 20000, 500
	tasks := make([]*Task, 0, count)
	doTestQueueImplAdd(t, "test-ok", count, func(queue QueueHandler, topic string, i int) {
		queue.AddWithKey(topic, strconv.Itoa(i+adding), strconv.Itoa(i+adding), 0)
	}, func(topic string, task *Task) (finished bool, retryAfter time.Duration, err error) {
		tasks = append(tasks, task)
		return true, 0, nil
	})
	if n := len(tasks); n != count {
		t.Error(fmt.Errorf("assert faild: expect %v, but %v", count, n))
	}
	for i := 0; i < count; i++ {
		s := strconv.Itoa(i + adding)
		var val *Task
		for _, item := range tasks {
			if item.Data == s && item.Key == s {
				val = item
				break
			}
		}
		if val == nil {
			t.Error(fmt.Errorf("task data %v not found", s))
		}
	}
}

// 测试失败后自动重试的机制
func TestQueue_Error(t *testing.T) {
	adding, count := 30000, 500
	tasks := make([]*Task, 0, count)
	minsAfter := time.Now().Add(5 * time.Second)
	doTestQueueImplAdd(t, "test-error", count, func(queue QueueHandler, topic string, i int) {
		queue.AddWithKey(topic, strconv.Itoa(i+adding), "", 0)
	}, func(topic string, task *Task) (finished bool, retryAfter time.Duration, err error) {
		if time.Now().After(minsAfter) {
			tasks = append(tasks, task)
			return true, 0, nil
		} else {
			return false, 3 * time.Second, nil
		}
	})
	if n := len(tasks); n != count {
		t.Error(fmt.Errorf("assert faild: expect %v, but %v", count, n))
	}
	for i := 0; i < count; i++ {
		s := strconv.Itoa(i + adding)
		var val *Task
		for _, item := range tasks {
			if item.Key == s {
				val = item
				break
			}
		}
		if val == nil {
			t.Error(fmt.Errorf("task data %v not found", s))
		}
	}
}

func doTestQueueImplAdd(t *testing.T, topic string, count int, addFunc func(queue QueueHandler, topic string, count int), handler QueueHandlerFunc) {
	queue := NewHandler(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), nil)
	added, handled := int32(0), int32(0)
	var start time.Time
	var err error

	start = time.Now()
	for i := 0; i < count; i++ {
		addFunc(queue, topic, i)
		atomic.AddInt32(&added, 1)
	}
	t.Log(fmt.Sprintf("add %v, timespan=%v", added, time.Now().Sub(start)))

	err = queue.RegisterHandler(topic, func(topic string, task *Task) (_ bool, _ time.Duration, _ error) {
		atomic.AddInt32(&handled, 1)
		return handler(topic, task)
	}, &HandlerOptions{
		Worker: 8,
	})
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
		if n, _ := queue.Count(topic, time.Now().Add(2400*time.Hour)); n == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	queue.Stop()

	t.Log(fmt.Sprintf("handled=%v, timespan=%v", handled, time.Now().Sub(start)))
}
