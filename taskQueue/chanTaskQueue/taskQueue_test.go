package chanTaskQueue

import (
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	handled, count := 0, 100
	queue := New("test", 1000, func(job interface{}, t time.Time) {
		time.Sleep(3 * time.Millisecond)
		handled++
	})
	queue.Start()

	start := time.Now()
	for i := 0; i < count; i++ {
		queue.Add(i)
	}
	t.Logf("start ts: %v", time.Now().Sub(start))

	start = time.Now()
	queue.Stop(0)
	t.Logf("stop ts: %v", time.Now().Sub(start))

	if handled != count {
		t.Errorf("assert faild: %v", handled)
	}
}

func TestQueueImpl_Pause(t *testing.T) {
	handled, count := 0, 100
	queue := New("test", 1000, func(job interface{}, t time.Time) {
		time.Sleep(3 * time.Millisecond)
		handled++
	})
	queue.Start()

	start := time.Now()
	for i := 0; i < count; i++ {
		queue.Add(i)
	}
	t.Logf("start ts: %v, count=%v", time.Now().Sub(start), count)

	time.Sleep(100 * time.Millisecond)
	queue.Pause()
	t.Logf("sleep and pause ts: %v, handled=%v", time.Now().Sub(start), handled)
	if d := time.Second; d != 0 {
		time.Sleep(d)
		n := handled
		queue.Resume()
		t.Logf("sleep and resume ts: %v, handled=%v", time.Now().Sub(start), n)
	}

	start = time.Now()
	queue.Stop(0)
	t.Logf("stop ts: %v, handled=%v", time.Now().Sub(start), handled)

	if handled != count {
		t.Error("assert faild")
	}
}
