// ------------------------------------------------------------------------------
// 基于时间周期的计数器，用来实现类似 “最近 N 个时间周期” 的性能记数。
// ------------------------------------------------------------------------------
package timeRoundedCounter

import (
	"github.com/emirpasic/gods/lists/singlylinkedlist"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"yelo/go-util/mathUtil"
)

// 按时间周期分组的计数器，比如设定技术周期为一分钟时，每次记数后会按照每分钟一个数据将其汇总。
type TimeRoundedCounter interface {
	// 获取计数器的 Name
	GetName() string
	// 获取计数器周期。计数器周期不可更改。
	GetRound() time.Duration
	// 获取最大保留的记数周期
	GetCycle() int
	// 设置最大保留的记数周期
	SetCycle(cycle int)
	// 记数
	Add(n int32)
	// 记数
	Set(n int32)
	// 获取记数结果，移除右侧的 0 值
	GetData() []int32
	// 获取记数结果。 count = -1 表示返回全部结果，rtrimZero 指示是否移除右侧的 0 值
	GetDataSlice(offset, count int, rtrimZero bool) []int32
}

// 创建一个计数器
func New(round time.Duration, cycle int) TimeRoundedCounter {
	return NewNamed("", round, cycle)
}

// 创建一个有名字的计数器，可以通过 GetNamedCounter 获取
func NewNamed(name string, round time.Duration, cycle int) TimeRoundedCounter {
	val := &counterImpl{
		id:    atomic.AddInt32(&counterId, 1),
		name:  name,
		round: round,
		cycle: cycle,
		items: singlylinkedlist.New(),
	}
	if val.name != "" {
		activeCounter[val.id] = val
	}
	return val
}

var (
	counterId         = int32(0)
	activeCounter     = make(map[int32]*counterImpl, 32)
	activeCounterLock = sync.RWMutex{}
)

// 获取所有 Name 不为空的计数器
func GetNamedCounter() []TimeRoundedCounter {
	activeCounterLock.RLock()
	defer activeCounterLock.RUnlock()
	arr, idx := make([]TimeRoundedCounter, len(activeCounter)), 0
	for _, v := range activeCounter {
		arr[idx] = v
		idx++
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].GetName() < arr[j].GetName()
	})
	return arr
}

type counterImpl struct {
	id    int32
	name  string
	round time.Duration
	cycle int
	items *singlylinkedlist.List
	lock  sync.RWMutex
}
type counterItem struct {
	unix  int64
	count int32
}

func (this *counterImpl) GetName() string { return this.name }

func (this *counterImpl) GetRound() time.Duration { return this.round }

func (this *counterImpl) GetCycle() int { return this.cycle }

func (this *counterImpl) SetCycle(cycle int) { this.cycle = cycle }

func (this *counterImpl) Add(n int32) {
	this.countNow(n, true, time.Now())
}

func (this *counterImpl) Set(n int32) {
	this.countNow(n, false, time.Now())
}

func (this *counterImpl) countNow(n int32, addOrSet bool, t time.Time) {
	if this.cycle <= 0 {
		return
	}

	unix := t.Truncate(this.round).UnixNano()

	this.lock.Lock()
	defer this.lock.Unlock()

	if val, ok := this.items.Get(0); ok {
		if item := val.(*counterItem); item.unix == unix {
			if addOrSet {
				item.count += n
			} else {
				item.count = n
			}
			return
		}
	}

	this.items.Prepend(&counterItem{unix: unix, count: n})
	if this.items.Size() > this.cycle {
		this.items.Remove(this.cycle)
	}
}

func (this *counterImpl) GetData() []int32 {
	return this.GetDataSlice(0, -1, true)
}

func (this *counterImpl) GetDataSlice(offset, count int, rtrimZero bool) []int32 {
	return this.getDataSliceNow(offset, count, rtrimZero, time.Now())
}

func (this *counterImpl) getDataSliceNow(offset, count int, rtrimZero bool, t time.Time) []int32 {
	if count == 0 || offset > this.cycle {
		return nil
	}
	if offset < 0 {
		offset = 0
	}

	if count > 0 {
		count = mathUtil.MinInt(this.cycle, count) - offset
	} else {
		count = this.cycle - offset
	}
	result := make([]int32, count)
	roundNs := int64(this.round)
	maxNs := t.Truncate(this.round).UnixNano()
	if offset > 0 {
		maxNs -= int64(offset) * roundNs
	}
	minNs := maxNs - int64(count-1)*int64(this.round)

	this.lock.RLock()
	defer this.lock.RUnlock()

	iter, notZeroIdx := this.items.Iterator(), -1
	for iter.Next() {
		item := iter.Value().(*counterItem)
		if item.unix < minNs {
			break
		} else if idx := int((maxNs - item.unix) / roundNs); idx >= 0 && idx < count {
			result[idx] = item.count
			if rtrimZero {
				if item.count != 0 {
					notZeroIdx = idx
				}
			}
		}
	}

	if rtrimZero && notZeroIdx < count-1 {
		result = result[:notZeroIdx+1]
	}

	return result
}
