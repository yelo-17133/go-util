package timeUtil

// ------------------------------------------------------------------------------
// 对计时器的一个扩展：
//    1、允许修改计时器的间隔。这样如果某个计时器的间隔是基于配置文件的，可以在配置文件变更后直接修改间隔。
//    2、允许手动触发一次 Tick、允许指定第一次触发 Tick 的延期时间（默认的计时器是一个计时周期）。有时候需要在创建计时器的时候立即或者在很短时间内就先触发一次，比如用于更新缓存的计时器。
// ------------------------------------------------------------------------------

import (
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"sync"
	"time"
	"yelo/go-util/runtimeUtil"
)

type Ticker interface {
	// 获取 Ticker 间隔
	GetDuration() time.Duration
	// 设置 Ticker 间隔
	SetDuration(d time.Duration)
	// 停止
	Stop(timeout time.Duration) bool
	// 手动触发一次 Tick 并更新下次触发时间
	Trigger()
}

type tickerImpl struct {
	ticker   *time.Ticker
	stopChan chan bool
	d        time.Duration
	f        func()
	next     int64
	running  bool
	lock     sync.RWMutex
}

func NewTicker(interval, after time.Duration, f func()) Ticker {
	return (&tickerImpl{d: interval, f: f, next: time.Now().UnixNano() + int64(after)}).restart()
}

func (this *tickerImpl) GetDuration() time.Duration {
	return this.d
}

func (this *tickerImpl) SetDuration(d time.Duration) {
	if d != this.d {
		this.d = d
		this.restart()
	}
}

func (this *tickerImpl) Trigger() {
	this.doTrigger(false)
}

func (this *tickerImpl) doTrigger(check bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if check {
		if now := time.Now().UnixNano(); now < this.next {
			return
		}
	}

	if e := runtimeUtil.CallFunc(this.f); e != nil {
		os.Stderr.WriteString(fmt.Sprintf("[%v] timeUtil.Ticker panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e))
		debug.PrintStack()
	}
	this.next += int64(this.d)
}

func (this *tickerImpl) Stop(timeout time.Duration) bool {
	if this.running {
		this.ticker.Stop()
		this.stopChan <- true

		end := time.Now().Add(timeout).UnixNano()
		for this.running && time.Now().UnixNano() < end {
			time.Sleep(time.Millisecond)
		}
	}
	return !this.running
}

func (this *tickerImpl) restart() *tickerImpl {
	this.Stop(time.Minute)
	this.ticker = time.NewTicker(this.d / 10)
	this.running = true
	this.stopChan = make(chan bool)
	go func() {
		defer func() {
			this.running = false
			close(this.stopChan)
		}()
		for {
			select {
			case <-this.ticker.C:
				this.doTrigger(true)
			case stop := <-this.stopChan:
				if stop {
					return
				}
			}
		}
	}()
	return this
}

func (this *tickerImpl) callF() {
	defer func() {
		if e := recover(); e != nil {
			os.Stderr.WriteString(fmt.Sprintf("[%v] timeUtil.Ticker panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e))
			debug.PrintStack()
		}
	}()
	tmp := this.next
	this.next = math.MaxInt64
	this.f()
	this.next = tmp + int64(this.d)
}
