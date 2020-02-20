// 基于 channel 实现的异步队列。
// 在异步处理数据时，一般针对每个数据启动一个协程来处理该数据。比如我们会监听某个消息队列，在消息队列中有新数据到达时就启动携程来处理他。
// 这种处理方式一般情况下不会有问题，但如果消息队列中的数据非常频繁、且每次数据处理耗时非常短，则此时会因为频繁的协程切换而导致浪费系统性能。协程的代价虽然比线程要少，但切换仍然有代价。
// 此时，一个更好的做法，就是使用 channel 处理，启动一个协程来处理该 chanel。这样可以避免频繁的协程切换。
// 同时，还可以通过控制 channel 大小来做服务容量限制，防止雪崩效应产生。
package chanTaskQueue

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"yelo/go-util/mathUtil"
	"yelo/go-util/osUtil"
	"yelo/go-util/timeRoundedCounter"
)

const (
	Status_Created  = 0
	Status_Running  = 1
	Status_Pause    = 2
	Status_Stopping = 3
	Status_Stopped  = 4
)

type Queue interface {
	// 获取队列名称
	Name() string
	// 获取容量
	Capacity() int
	// 获取当前大小
	Size() int
	// 获取当前工作线程数量
	Worker() int
	// 设置当前工作线程数量
	SetWorker(worker int) error
	// 获取当前状态
	Status() Status
	// 获取创建队列时指定的计数器
	Counter() timeRoundedCounter.TimeRoundedCounter
	// 入队，如果队列已满则返回 false。否则返回 true。
	Add(job interface{}) (bool, Status)
	// 启动处理程序
	Start() error
	// 暂停
	Pause() error
	// 恢复
	Resume() error
	// 停止处理，参数指定超时时间，并返回是否已经成功停止。
	Stop(timeout time.Duration) (bool, Status)
	// 强制中止，丢弃未完成的任务
	Abort()
}

type Status int

type HandlerFunc func(job interface{}, t time.Time)

type Options struct {
	Worker  int
	Counter timeRoundedCounter.TimeRoundedCounter
}

func New(name string, capacity int, handler HandlerFunc, opt ...*Options) Queue {
	if handler == nil {
		panic("参数 handler 不能为空")
	}

	var realOpt *Options
	if len(opt) != 0 && opt[0] != nil {
		realOpt = opt[0]
	} else {
		realOpt = &Options{}
	}
	realOpt.Worker = mathUtil.MinMaxInt(realOpt.Worker, 1, 1024)

	this := &queueImpl{
		id:       atomic.AddInt32(&queueId, 1),
		name:     name,
		capacity: capacity,
		worker:   realOpt.Worker,
		jobChan:  make(chan interface{}, capacity+64),
		handler:  handler,
		status:   Status_Created,
		counter:  realOpt.Counter,
	}
	return this
}

type queueImpl struct {
	runningWorker int32 // 需要 atomic 原子操作，放在结构体开头保证内存地址对齐
	id            int32
	name          string
	capacity      int
	worker        int
	jobChan       chan interface{}
	handler       func(job interface{}, t time.Time)
	waitGroup     sync.WaitGroup
	stopChan      []chan bool
	currJob       *jobWrap
	status        Status
	statusLock    sync.Mutex
	counter       timeRoundedCounter.TimeRoundedCounter
}

type jobWrap struct {
	job  interface{}
	time time.Time
}

var (
	queueId         = int32(1)
	activeQueue     = make(map[int32]*queueImpl, 32)
	activeQueueLock = sync.RWMutex{}
)

func init() {
	osUtil.OnSignalExit(func(sig os.Signal) {
		activeQueueLock.RLock()
		for _, val := range activeQueue {
			val.Stop(5 * time.Second)
		}
		activeQueueLock.RUnlock()
	})
}

func GetActiveQueue() []Queue {
	activeQueueLock.RLock()
	defer activeQueueLock.RUnlock()
	arr, idx := make([]Queue, len(activeQueue)), 0
	for _, v := range activeQueue {
		arr[idx] = v
		idx++
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Name() < arr[j].Name()
	})
	return arr
}

func setActive(id int32, val *queueImpl) {
	activeQueueLock.Lock()
	defer activeQueueLock.Unlock()
	if tmp := activeQueue[id]; tmp != val {
		if val == nil {
			delete(activeQueue, id)
		} else {
			activeQueue[id] = val
		}
	}
}

func (this Status) String() string {
	switch this {
	case Status_Created:
		return "created"
	case Status_Running:
		return "running"
	case Status_Stopping:
		return "stopping"
	case Status_Stopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// 获取容量
func (this *queueImpl) Name() string { return this.name }

// 获取容量
func (this *queueImpl) Capacity() int { return this.capacity }

// 获取当前大小
func (this *queueImpl) Size() int { return len(this.jobChan) }

// 获取处理程序数量
func (this *queueImpl) Worker() int { return len(this.stopChan) }

// 获取当前是否正在运行（已启动、尚未停止）
func (this *queueImpl) Status() Status { return this.status }

// 获取创建队列时指定的计数器
func (this *queueImpl) Counter() timeRoundedCounter.TimeRoundedCounter { return this.counter }

// 入队。返回是否成功，以及队列状态。如果队列处于 Status_Stopping|Status_Stopped ，或者队列已满，都会导致入队失败。
func (this *queueImpl) Add(job interface{}) (bool, Status) {
	if this.status == Status_Stopping || this.status == Status_Stopped {
		return false, this.status
	}
	if len(this.jobChan) >= this.capacity {
		return false, this.status
	}
	this.waitGroup.Add(1)
	this.jobChan <- &jobWrap{job: job, time: time.Now()}
	if this.counter != nil {
		this.counter.Add(1)
	}
	if this.status == Status_Created {
		setActive(this.id, this)
	}
	return true, this.status
}

// 启动处理程序
func (this *queueImpl) Start() error {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()

	if this.status == Status_Running || this.status == Status_Pause {
		return nil
	} else if this.status != Status_Created {
		return fmt.Errorf("队列已停止或正在停止[%v]", this.status.String())
	}

	this.ensureWorker()
	this.status = Status_Running
	setActive(this.id, this)

	return nil
}

func (this *queueImpl) Pause() error {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()
	if this.status == Status_Running {
		for this.currJob != nil {
			time.Sleep(20 * time.Millisecond)
		}
		this.status = Status_Pause
	}
	return nil
}

func (this *queueImpl) Resume() error {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()
	if this.status == Status_Pause {
		this.status = Status_Running
	}
	return nil
}

func (this *queueImpl) SetWorker(worker int) error {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()

	this.worker = worker
	if this.status == Status_Running || this.status == Status_Pause {
		this.ensureWorker()
	}

	return nil
}

func (this *queueImpl) ensureWorker() {
	for i, n := len(this.stopChan), mathUtil.MaxInt(1, this.worker); i < n; i++ {
		this.stopChan = append(this.stopChan, make(chan bool))
		atomic.AddInt32(&this.runningWorker, 1)
		go func(i int) {
			for {
				select {
				case v := <-this.jobChan:
					if v != nil {
						for this.status == Status_Pause {
							time.Sleep(20 * time.Millisecond)
						}
						if w, ok := v.(*jobWrap); ok {
							this.currJob = w
							this.handler(w.job, w.time)
							this.waitGroup.Add(-1)
							this.currJob = nil
						}
					} else if this.status == Status_Stopped {
						return
					}
				case v := <-this.stopChan[i]:
					if v && atomic.AddInt32(&this.runningWorker, -1) == 0 {
						go func() {
							this.waitGroup.Wait()
							this.doStop()
						}()
					}
				}
			}
		}(i)
	}
}

func (this *queueImpl) doStop() {
	this.status = Status_Stopped
	for _, c := range this.stopChan {
		close(c)
	}
	close(this.jobChan)
	setActive(this.id, nil)
}

// 停止处理，参数指定超时时间，并返回是否已经成功停止。
func (this *queueImpl) Stop(timeout time.Duration) (bool, Status) {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()

	if this.status == Status_Created || this.status == Status_Stopped {
		return true, this.status
	}

	if this.status == Status_Running || this.status == Status_Pause {
		this.status = Status_Stopping
		go func() {
			for _, c := range this.stopChan {
				c <- true
			}
		}()
	}

	sleep := 10 * time.Microsecond
	if timeout > 0 {
		for end := time.Now().Add(timeout); this.status != Status_Stopped && time.Now().Before(end); {
			time.Sleep(sleep)
		}
		return this.status == Status_Stopped, this.status
	} else {
		for this.status != Status_Stopped {
			time.Sleep(sleep)
		}
		return true, this.status
	}
}

func (this *queueImpl) Abort() {
	this.statusLock.Lock()
	defer this.statusLock.Unlock()

	if this.status == Status_Created || this.status == Status_Stopped {
		return
	}

	this.doStop()
}
