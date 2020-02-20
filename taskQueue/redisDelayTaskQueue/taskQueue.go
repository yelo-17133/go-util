// ------------------------------------------------------------------------------
// 这是一个利用 Redis 实现的延迟队列。可实现：
//   1、将任务放入队列并制定延迟一段时间之后再执行；
//   2、当某个任务处理失败时，不抛弃而是将其重新放入队列的后面并延迟一段时间执行、同时继续处理下一个。这样当某一个任务处理失败时不会阻塞。
//      一些有周期性的任务也可以通过该队列实现，比如针对每个用户每隔5分钟刷新一下数据。可以在每次刷新数据后将任务重新丢回到队列里并延迟五分钟。
//   3、自动保存重试次数和上次的错误消息。
//   4、支持 N-N 读写队列，即可以同时为同一个 topic 创建多个生产者和多个消费者（消费者使用分布式锁实现互斥）。
//
// 队列有两个 Interface：
// Queue:
//   如果只需要向队列中添加任务而不需要处理，可以使用该接口。相当于生产者。
//   创建任务时，会将任务数据存放在一个 Hash 中，同时将任务按照延迟时间放入 ZSet，利用 ZSet 实现延迟。
// QueueHandler：
//   如果需要处理队列中的任务，则使用该接口。该接口也同时继承了 Queue 的所有操作。相当于消费者+生产者。
//   Handler 需要指定一个任务处理的回调函数以及重试策略，如果回调函数返回 error 时则表示该任务处理失败了，此时会根据重试策略把任务重新丢回延迟队列。
//   Handler 会不断尝试从 ZSet 里面取出一个已经到期的任务并处理，期间会通过分布式锁实现互斥。
//
// 性能可运行 TestQueue 获得。开发环境测试结果为：生产者大约 2000 每秒、消费者大约 500 每秒（使用了分布式锁）
//
// 使用方法解释如下：
//   有时候，我们会需要程序在一段时间之后执行某个逻辑。
//   比如 “用户登录满10分钟送xxx礼包”  等，我们可以在用户登录那一刻启动一个 10min 的计时器，计时器到达时给用户送礼包。
//   但这样的实现方案有个很大的弊端：在计时器触发之前，一旦进程被重启，则会因为计时器丢失，这个逻辑就再也不会被触发了。
//   因此，正确的做法，应当是在用户登录那一刻，在系统的某个地方标记 10min 后需要给用户送礼包、同时启动一个扫描逻辑，不断的扫描截至当前时间有没有 “需要送礼包但尚未送” 的用户，如果有则送给他并标记为已送。
//   延迟队列，就是这样的一种允许指定延时的任务队列：当任务被丢到队列时，可以指定延迟多长时间，如果该时间大于0，则只有到达指定时间时该任务才会被发送到队列消费者那里。
//   如果所有任务在丢到队列里面时的延迟都是0，则延迟队列退化为普通队列，redisDelayTaskQueue 就变成了 redisTaskQueue。
// 延迟队列（redisDelayTaskQueue ）与普通队列（redisTaskQueue）的另一个不同之处在于：
//   在普通队列中，任务是没有唯一标识的，同样的数据如果向队列中放入两次，则会被消费两次。
//   延迟队列支持给任务设置唯一ID：如果同一个任务向队列中放入两次，则实际上后放入的数据会覆盖前一次的数据，从而实现一个任务只会被处理一次。
//   例如：
//     如果需要“用户离线超过3天则向其发送通知”。
//     当用户在t1登出系统时，可以向队列中增加一个通知任务，在t1+3day时触发。
//     但第二天用户又登录了系统，并在t2时登出，则再次向队列中增加此任务（相同的任务ID），设为t2+3day触发。
//     此时队列中仍然只有一个t2+3day的任务，在t1+3day的时候不会有任何事情发生。因为第二次增加任务时，实际上是修改了任务的执行时间，而不是增加了一个新的任务。
// ------------------------------------------------------------------------------
package redisDelayTaskQueue

import (
	"fmt"
	"github.com/go-redis/redis"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	"yelo/go-util/deepcopy"
	"yelo/go-util/jsonUtil"
	"yelo/go-util/osUtil"
	"yelo/go-util/redisLock"
	"yelo/go-util/strUtil"
	"yelo/go-util/timeRoundedCounter"
	"yelo/go-util/timeUtil"
)

type Queue interface {
	// 获取 Redis 客户端
	RedisClient() *redis.Client
	// 设置 Redis 客户端
	SetRedisClient(client *redis.Client) error
	// 新增一个任务，参数指定推迟多长时间调度。0 表示立即调度
	Add(topic, data string, after time.Duration) error
	// 新增一个任务，如果已经存在相同的 Key 则会覆盖之前的任务数据及推迟时间。参数指定推迟多长时间调度
	AddWithKey(topic, key, data string, after time.Duration) error
	// 删除一个任务
	DelWithKey(topic, key string) error
	// 获取指定任务分组中截至指定时间的待处理任务数量
	Count(topic string, now time.Time) (int, error)
	// 根据 key 获取任务的下次调度时间，如果出错或者 key 不存在，则返回 ZeroTime
	GetNextTime(topic, key string) (time.Time, error)
}

type QueueHandler interface {
	Queue

	// 获取 Handler 计数器，在创建时指定。该计数器按时间周期统计最近处理过的任务个数。
	HandlerCounter() timeRoundedCounter.TimeRoundedCounter
	// 根据任务分组注册任务处理回调函数以及出错后的重试策略
	//   topic: 队列名称
	//   handler: 任务处理回调函数
	RegisterHandler(topic string, handler QueueHandlerFunc, opt ...*HandlerOptions) error
	// 启动任务处理程序
	Start(opt ...Options) error
	// 停止任务处理程序
	Stop()
}

// 任务处理的回调函数。
// param:
//   topic: 任务分组
//   task: 任务状态信息
//   createTime: 任务创建时间
//   retried: 任务已经重试过的次数
// return:
//   finished: 任务是否已经处理完毕。如果为 true，则该任务会从队列中移除。
//   retryDelay: 当 finished=false 时，指示任务需要延长多长时间重试。最小不低于 3 秒，小于该值会被更正为 3 秒。
//   err: 任务处理过程中的错误
type QueueHandlerFunc func(topic string, task *Task) (finished bool, retryAfter time.Duration, err error)

type Options struct {
	CheckInterval     time.Duration `json:"checkInterval,omitempty"`
	HandleTimeout     time.Duration `json:"handleTimeout,omitempty"`
	DefaultRetryAfter time.Duration `json:"defaultRetryAfter,omitempty"`
}

type HandlerOptions struct {
	Worker  int                 `json:"worker,omitempty"`
	OnPanic func(e interface{}) `json:"-"`
}

type Task struct {
	Key       string  `json:"key,omitempty" description:"任务数据的 Key"`
	Data      string  `json:"data,omitempty" description:"任务数据"`
	Error     string  `json:"error,omitempty" description:"（最后一次）处理任务时发生的错误"`
	ErrorTime float64 `json:"errorTime,omitempty" description:"（最后一次）处理任务发生错误时的时间（秒，浮点数）"`
	Retried   int     `json:"retried,omitempty" description:"已重试的次数"`
}

// 创建一个 Queue 实例
// 参数:
//   client: Redis 客户端
func New(client *redis.Client) Queue {
	return &queueImpl{
		client:     client,
		handlerMap: make(map[string]*topicHandlerWrap),
		redisLock:  redisLock.New(client, "DelayTaskQueue_"),
	}
}

// 创建一个 QueueHandler 实例
// 参数:
//   client: Redis 客户端
//   handlerCounter: 用于统计已处理的任务数的计数器，nil 表示不统计
func NewHandler(client *redis.Client, handlerCounter timeRoundedCounter.TimeRoundedCounter) QueueHandler {
	return &queueImpl{
		client:     client,
		handlerMap: make(map[string]*topicHandlerWrap),
		redisLock:  redisLock.New(client, "DelayTaskQueue_"),
		counter:    handlerCounter,
	}
}

type queueImpl struct {
	client     *redis.Client
	opt        Options
	handlerMap map[string]*topicHandlerWrap
	redisLock  redisLock.RedisLock
	lock       sync.RWMutex
	once       sync.Once
	counter    timeRoundedCounter.TimeRoundedCounter
	status     int // 状态: 0=Created; 1=Running; 2=Stopping; 4=Stopped
}

const (
	redisKeyPrefix = "DelayTaskQueue:"
	minRetryAfter  = 3 * time.Second
)

type topicHandlerWrap struct {
	handler QueueHandlerFunc
	opt     *HandlerOptions
	ticker  []*time.Ticker
	stop    []chan bool
}

func (this *queueImpl) RedisClient() *redis.Client {
	return this.client
}

func (this *queueImpl) SetRedisClient(client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("参数不能为空")
	}
	if err := client.Ping().Err(); err != nil {
		return fmt.Errorf("访问 Redis 失败: %v", err)
	}
	this.client = client
	this.redisLock.SetRedisClient(client)
	return nil
}

func (this *queueImpl) Add(topic, data string, after time.Duration) error {
	return this.AddWithKey(topic, strUtil.Rand(6), data, after)
}

func (this *queueImpl) prepareCmd(topic string) (formattedTopic, redisKeyQueue, redisKeyData string, err error) {
	if this.client == nil {
		return "", "", "", fmt.Errorf("必须先设置 Redis Client")
	}

	formattedTopic = strings.Replace(topic, ":", "-", -1)
	if formattedTopic == "" {
		formattedTopic = "Default"
	}

	redisKeyQueue = redisKeyPrefix + topic + ":Queue"
	redisKeyData = redisKeyPrefix + topic + ":Data"

	return
}

// Add With Key
func (this *queueImpl) AddWithKey(topic, key, data string, after time.Duration) error {
	topic, redisKeyQueue, redisKeyData, err := this.prepareCmd(topic)
	if err != nil {
		return err
	}

	// 先写 Hash、后写 ZSet
	if data != "" {
		if err := this.client.HSet(redisKeyData, key, jsonUtil.MustMarshalToString(&Task{Data: data})).Err(); err != nil && err != redis.Nil {
			return err
		}
	} else {
		this.client.HDel(redisKeyData, key)
	}

	scoreTime := time.Now()
	if after > 0 {
		scoreTime = scoreTime.Add(after)
	}
	if err := this.client.ZAdd(redisKeyQueue, redis.Z{Score: timeUtil.ToSecondFloat(scoreTime, 6), Member: key}).Err(); err != nil && err != redis.Nil {
		// 如果出错，把 Data 中的元素也删了。这里即使操作失败也没关系，最多只会有脏数据、不会导致逻辑错误。
		if data != "" {
			this.client.HDel(redisKeyData, key)
		}
		return err
	}

	return nil
}

func (this *queueImpl) DelWithKey(topic, key string) error {
	_, redisKeyQueue, redisKeyData, err := this.prepareCmd(topic)
	if err != nil {
		return err
	}

	if n, err := this.client.ZRem(redisKeyQueue, key).Result(); err != nil && err != redis.Nil {
		return err
	} else if n > 0 {
		this.client.HDel(redisKeyData, key)
	}

	return nil
}

func (this *queueImpl) Count(topic string, now time.Time) (int, error) {
	_, redisKeyQueue, _, err := this.prepareCmd(topic)
	if err != nil {
		return 0, err
	}

	max := strconv.FormatFloat(timeUtil.ToSecondFloat(now, 6), 'f', -1, 64)
	count, err := this.client.ZCount(redisKeyQueue, "-inf", max).Result()
	if err != nil && err != redis.Nil {
		return 0, err
	}

	return int(count), nil
}

func (this *queueImpl) GetNextTime(topic, key string) (time.Time, error) {
	_, redisKeyQueue, _, err := this.prepareCmd(topic)
	if err != nil {
		return time.Time{}, err
	}

	score, err := this.client.ZScore(redisKeyQueue, key).Result()
	if err == nil {
		return timeUtil.FromSecondFloat(score), nil
	} else if err == redis.Nil {
		return time.Time{}, nil
	} else {
		return time.Time{}, err
	}
}

// 获取 Handler 计数器
func (this *queueImpl) HandlerCounter() timeRoundedCounter.TimeRoundedCounter {
	return this.counter
}

func (this *queueImpl) RegisterHandler(topic string, handler QueueHandlerFunc, opt ...*HandlerOptions) error {
	if handler == nil {
		return fmt.Errorf("参数 handler 不能为空")
	}

	var realOpt *HandlerOptions
	if len(opt) != 0 {
		realOpt = opt[0]
	} else {
		realOpt = &HandlerOptions{}
	}
	if realOpt.Worker <= 0 {
		realOpt.Worker = 1
	}

	topic = strings.Replace(topic, ":", "-", -1)
	if topic == "" {
		topic = "Default"
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.status != 0 {
		return fmt.Errorf("请在启动前调用")
	}

	this.handlerMap[topic] = &topicHandlerWrap{handler: handler, opt: realOpt}

	return nil
}

func (this *queueImpl) Start(opt ...Options) error {
	if this.status != 0 {
		return nil
	}

	if this.client == nil {
		return fmt.Errorf("必须先设置 Redis Client")
	}

	// 设置参数
	if len(opt) != 0 {
		this.opt = opt[0]
	} else {
		this.opt = Options{}
	}
	if this.opt.CheckInterval <= 0 {
		this.opt.CheckInterval = 250 * time.Millisecond
	}
	if this.opt.HandleTimeout <= 0 {
		this.opt.HandleTimeout = 10 * time.Second
	}
	if this.opt.DefaultRetryAfter <= 0 {
		this.opt.DefaultRetryAfter = 10 * time.Second
	}
	if this.opt.DefaultRetryAfter < minRetryAfter {
		this.opt.DefaultRetryAfter = minRetryAfter
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.status != 0 {
		return nil
	}

	for topic, handler := range this.handlerMap {
		this.startTopicHandler(topic, handler)
	}

	this.status = 1

	osUtil.OnSignalExit(func(sig os.Signal) {
		this.Stop()
	})

	return nil
}

func (this *queueImpl) startTopicHandler(topic string, handlerWrap *topicHandlerWrap) {
	redisKeyData := redisKeyPrefix + topic + ":Data"
	redisKeyQueue := redisKeyPrefix + topic + ":Queue"
	handlerWrap.ticker = make([]*time.Ticker, handlerWrap.opt.Worker)
	handlerWrap.stop = make([]chan bool, handlerWrap.opt.Worker)
	for i := 0; i < handlerWrap.opt.Worker; i++ {
		handlerWrap.ticker[i], handlerWrap.stop[i] = time.NewTicker(this.opt.CheckInterval), make(chan bool)
		go func(ticker *time.Ticker, stop chan bool) {
			for {
				select {
				case <-stop:
					ticker.Stop()
					return
				case <-ticker.C:
					if this.client == nil {
						continue
					}

					now, n := time.Now(), 0
					timeout := now.Add(this.opt.HandleTimeout)
					for time.Now().Before(timeout) {
						n++
						handled, err := this.fetchOneTask(timeUtil.ToSecondFloat(now, 6), topic, redisKeyQueue, redisKeyData, handlerWrap)
						if err != nil || !handled || n >= 100 {
							break
						}
					}
				}
			}
		}(handlerWrap.ticker[i], handlerWrap.stop[i])
	}
}

func (this *queueImpl) fetchOneTask(now float64, topic, redisKeyQueue, redisKeyData string, handler *topicHandlerWrap) (taskHandled bool, redisError error) {
	// 加锁
	ok, err := this.redisLock.Lock(topic, time.Minute, this.opt.HandleTimeout)
	if err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	// 弹出队列中的第一个任务，根据时间（score）判断是否已到期，如果没有任务或者第一个任务尚未到期则返回
	z, err := this.client.ZRangeWithScores(redisKeyQueue, 0, 0).Result()
	if err != nil && err != redis.Nil {
		this.redisLock.Unlock(topic)
		return false, err
	} else if len(z) == 0 {
		// 队列为空，则对应的哈希也应该为空。此处按概率，平均1分钟执行一次
		rand.Seed(time.Now().UnixNano())
		if rand.Intn(int(time.Second/this.opt.CheckInterval)*10) < 10 {
			this.client.Del(redisKeyData)
		}
		this.redisLock.Unlock(topic)
		return false, nil
	} else if z[0].Score >= now {
		// 队列非空，但执行时间都晚于当前时间，直接返回，轮空
		this.redisLock.Unlock(topic)
		return false, nil
	}

	// 找到了一个需要处理的任务。此处直接从队列删除，后面如果处理出错会再次写回队列
	this.client.ZRem(redisKeyQueue, z[0].Member)
	this.redisLock.Unlock(topic)

	key := z[0].Member.(string)
	task, retryAfter := &Task{Key: key}, time.Duration(0)

	finished := true
	defer func() {
		// 如果已经处理了任务（包括任务数据不合法无法处理），则删除任务数据
		if finished {
			this.client.HDel(redisKeyData, key)
		}

		// 计数器
		if key != "" && this.counter != nil {
			this.counter.Add(1)
		}
	}()

	// 获取 task 数据并反序列化
	taskStr, err := this.client.HGet(redisKeyData, key).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if taskStr != "" {
		if err := jsonUtil.UnmarshalFromString(taskStr, task); err != nil {
			// 忽略 json 反序列化错误，直接丢弃数据
			return true, nil
		}
	}

	// 执行回调函数，用闭包包裹以便即使回调函数出现 panic 仍能继续运行
	func() {
		defer func() {
			if e := recover(); e != nil {
				finished, task.Error = false, fmt.Sprintf("redisDelayTaskQueue.topicHandler[%s] panic: %v", topic, e)
				if handler.opt.OnPanic != nil {
					handler.opt.OnPanic(e)
				} else {
					os.Stderr.WriteString(fmt.Sprintf("[%v] redisDelayTaskQueue.topicHandler[%s] panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), topic, e))
					debug.PrintStack()
				}
			}
		}()
		if finished, retryAfter, err = handler.handler(topic, task); err != nil {
			task.Error = err.Error()
		} else {
			task.Error = ""
		}
	}()

	// 更新任务数据，重新放回队列
	if !finished {
		if retryAfter <= 0 {
			retryAfter = this.opt.DefaultRetryAfter
		} else if retryAfter < minRetryAfter {
			retryAfter = minRetryAfter
		}
		if task.Error != "" {
			task.Retried++
			task.ErrorTime = now
		} else {
			task.Retried = 0
			task.ErrorTime = 0
		}

		// update redisKeyData
		taskCopy := deepcopy.Copy(task).(*Task)
		taskCopy.Key = ""
		copyStr := jsonUtil.MustMarshalToString(taskCopy)
		if copyStr == "{}" {
			copyStr = ""
		}
		if copyStr != taskStr {
			if copyStr != "" {
				this.client.HSet(redisKeyData, key, copyStr)
			} else {
				this.client.HDel(redisKeyData, key)
			}
		}
		this.client.ZAdd(redisKeyQueue, redis.Z{Score: now + float64(retryAfter)/float64(time.Second), Member: key})
	}

	return true, nil
}

func (this *queueImpl) Stop() {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.status != 1 {
		return
	}
	this.status = 2

	for _, handler := range this.handlerMap {
		for i, ticker := range handler.ticker {
			ticker.Stop()
			handler.stop[i] <- true
		}
	}

	this.status = 3
}
