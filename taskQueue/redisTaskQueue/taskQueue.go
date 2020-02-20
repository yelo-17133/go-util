// ------------------------------------------------------------------------------
// 利用 Redis 实现的任务队列。可实现：
//   1、实现分布式任务队列。
//   2、消费者从尝试队列中取出一个消息、并调用指定的方法处理消息。
//      可以通过方法的返回值告诉队列该消息是否已经成功处理（根据该变量决定是否从队列中弹出此消息）
//      可以通过方法的返回值告诉队列出错后需要暂停多长时间重试（此时不弹出当前消息而是延迟指定时间重试）。
//
// 与 redisDelayTaskQueue 不同的是：
//   1、此队列当某个任务处理失败时，队列会阻塞在当前任务上一直等待、直到处理成功。
//   2、性能上，redisDelayTaskQueue 所有的 Handler 共同竞争同一个分布式锁、且每次加锁期间需要访问多个数据结构，锁冲突概率较高，顾性能稍慢。
//      但 redisQueue 每个 Handler 携程竞争各自的锁、每次加锁冲突概率小，且每次加锁期间只需要访问一个 List ，性能稍快。
// 可运行 TestQueue 性能结果。开发环境测试结果为：生产者大约 2000 每秒、消费者大约 500 每秒（使用了分布式锁）
// ------------------------------------------------------------------------------
package redisTaskQueue

import (
	"fmt"
	"github.com/go-redis/redis"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	"yelo/go-util/osUtil"
	"yelo/go-util/redisLock"
	"yelo/go-util/timeRoundedCounter"
)

type Queue interface {
	// 获取 Redis 客户端
	RedisClient() *redis.Client
	// 设置 Redis 客户端
	SetRedisClient(client *redis.Client) error
	// 新增一个任务
	Add(topic, data string) error
	// 获取指定任务分组中的待处理任务数量
	Count(topic string) (int, error)
}

type QueueHandler interface {
	// 获取 Redis 客户端
	RedisClient() *redis.Client
	// 设置 Redis 客户端
	SetRedisClient(client *redis.Client) error
	// 新增一个任务
	Add(topic, data string) error
	// 获取指定任务分组中的待处理任务数量
	Count(topic string) (int, error)
	// 获取 Handler 计数器，在创建时指定。该计数器按时间周期统计最近处理过的任务个数。
	HandlerCounter() timeRoundedCounter.TimeRoundedCounter
	// 根据任务分组注册任务处理回调函数
	RegisterHandler(topic string, handler QueueHandlerFunc, opt ...*HandlerOptions) error
	// 启动任务处理程序
	Start() error
	// 停止任务处理程序
	Stop()
}

// 任务处理的回调函数。
// param:
//   topic: 任务分组
//   data: 任务数据
// return:
//   success: 任务是否已经成功处理。如果为 true，则该任务会从队列中移除。
//   retryDelay: 当 finished=false 时，指示任务需要延长多长时间重试。最小不低于 3 秒，小于该值会被更正为 3 秒。
type QueueHandlerFunc func(topic string, data string) (success bool)

type HandlerOptions struct {
	// 要启动的工作协程数，默认为 1。
	Worker int
	// 在处理任务队列时，如果队列处理完毕或者已达到没批次最多处理任务数，等待多久再次检查队列。默认 1 秒。
	Interval time.Duration
	// 在处理任务队列时，每批次最多处理的任务数，超过该数量则直接休息 Interval 。默认 100 。
	// 如果 BatchCount 过大，会导致处理函数一直在处理任务而长时间独占 CPU，可能导致其他关键协程无法得到及时处理。
	// 如果 BatchCount 过小且 QueueHandlerFunc 很轻量，则会导致处理函数执行一个很小的任务之后就立刻休息 Interval，可能导致队列堆积。
	// 请根据队列处理程序 QueueHandlerFunc 的实际负载情况，合理设置 Interval 和 BatchCount 。
	BatchCount int
	// 在处理任务队列时，如果发生 Redis 错误或者 QueueHandlerFunc 处理失败，等待多久重试。默认 5 秒。
	// 该值不能设置太小，因为大概率情况下在很短时间之后重试还是会发生同样的错误。
	ErrorInterval time.Duration
	// 单个任务的处理超时时间（QueueHandlerFunc 最大执行时间），默认 5 秒。
	Timeout time.Duration
}

func New(client *redis.Client, opt ...*Options) Queue {
	return NewHandler(client, nil, opt...)
}

func NewHandler(client *redis.Client, handlerCounter timeRoundedCounter.TimeRoundedCounter, opt ...*Options) QueueHandler {
	var realOpt *Options
	if len(opt) != 0 {
		realOpt = opt[0]
	}
	if realOpt == nil {
		realOpt = &DefaultOptions
	}
	if realOpt.RedisRoot == "" {
		realOpt.RedisRoot = DefaultOptions.RedisRoot
	}

	return &queueImpl{
		client:         client,
		handlerMap:     make(map[string]*topicHandlerWrap),
		redisKeyPrefix: realOpt.RedisRoot + ":",
		redisLock:      redisLock.New(client, "TaskQueue_"),
		counter:        handlerCounter,
	}
}

type Options struct {
	RedisRoot string
}

var DefaultOptions = Options{
	RedisRoot: "TaskQueue",
}

type queueImpl struct {
	client         *redis.Client
	handlerMap     map[string]*topicHandlerWrap
	redisKeyPrefix string
	redisLock      redisLock.RedisLock
	lock           sync.RWMutex
	once           sync.Once
	counter        timeRoundedCounter.TimeRoundedCounter
	status         int // 状态: 0=Created; 1=Running; 2=Stopping; 4=Stopped
}

type topicHandlerWrap struct {
	HandlerOptions
	handler QueueHandlerFunc
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

func (this *queueImpl) Add(topic, data string) error {
	if this.client == nil {
		return fmt.Errorf("必须先设置 Redis Client")
	}

	topic = strings.Replace(topic, ":", "-", -1)
	if topic == "" {
		topic = "Default"
	}

	if err := this.client.LPush(this.redisKeyPrefix+topic+":Queue", data).Err(); err != nil && err != redis.Nil {
		return err
	}

	return nil
}

func (this *queueImpl) Count(topic string) (int, error) {
	if this.client == nil {
		return 0, fmt.Errorf("必须先设置 Redis Client")
	}

	topic = strings.Replace(topic, ":", "-", -1)
	if topic == "" {
		topic = "Default"
	}

	count, err := this.client.LLen(this.redisKeyPrefix + topic + ":Queue").Result()
	if err != nil && err != redis.Nil {
		return 0, err
	}

	return int(count), nil
}

// 获取 Handler 计数器
func (this *queueImpl) HandlerCounter() timeRoundedCounter.TimeRoundedCounter {
	return this.counter
}

func (this *queueImpl) RegisterHandler(topic string, handler QueueHandlerFunc, opt ...*HandlerOptions) error {
	if handler == nil {
		return fmt.Errorf("参数 handler 不能为空")
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

	handlerWrap := &topicHandlerWrap{handler: handler}
	if len(opt) != 0 && opt[0] != nil {
		handlerWrap.HandlerOptions = *opt[0]
	}
	if handlerWrap.Worker <= 0 {
		handlerWrap.Worker = 1
	}
	if handlerWrap.Interval <= 0 {
		handlerWrap.Interval = time.Second
	}
	if handlerWrap.BatchCount <= 0 {
		handlerWrap.BatchCount = 100
	}
	if handlerWrap.ErrorInterval <= 0 {
		handlerWrap.ErrorInterval = 5
	}
	if handlerWrap.Timeout <= 0 {
		handlerWrap.Timeout = 5 * time.Second
	}
	this.handlerMap[topic] = handlerWrap

	return nil
}

func (this *queueImpl) Start() error {
	if this.client == nil {
		return fmt.Errorf("必须先设置 Redis Client")
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.status != 0 {
		return nil
	}

	for topic, handler := range this.handlerMap {
		if err := this.startTopicHandler(topic, handler); err != nil {
			return err
		}
	}

	this.status = 1

	osUtil.OnSignalExit(func(sig os.Signal) {
		this.Stop()
	})

	return nil
}

func (this *queueImpl) startTopicHandler(topic string, handlerWrap *topicHandlerWrap) error {
	redisKeyQueue := this.redisKeyPrefix + topic + ":Queue"
	handlerWrap.ticker = make([]*time.Ticker, handlerWrap.Worker)
	handlerWrap.stop = make([]chan bool, handlerWrap.Worker)
	for i := 0; i < handlerWrap.Worker; i++ {
		handlerWrap.ticker[i], handlerWrap.stop[i] = time.NewTicker(handlerWrap.Interval), make(chan bool)
		go func(i int, ticker *time.Ticker, stop chan bool) {
			lockName := topic + ":Handler-" + strconv.Itoa(i)
			redisKeyHandling := this.redisKeyPrefix + lockName
			nextTime := int64(0)
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					if this.client == nil {
						continue
					}

					now := time.Now().UnixNano()
					if now < nextTime {
						continue
					}

					handledCount := 0
					for this.status == 1 {
						if handledCount++; handledCount >= handlerWrap.BatchCount {
							// 达到 BatchCount，直接进入下一个 Ticker
							nextTime = now + int64(handlerWrap.Interval)
							break
						}

						handled, success, err := this.tryDoOneTask(lockName, topic, redisKeyQueue, redisKeyHandling, handlerWrap)
						if err != nil {
							// redis 报错，按出错间隔暂停
							nextTime = now + int64(handlerWrap.ErrorInterval)
							break
						} else if !handled {
							// 队列空，直接进入下一个 Ticker
							nextTime = now + int64(handlerWrap.Interval)
							break
						} else if !success {
							// 队列非空、任务处理失败，按出错间隔暂停
							nextTime = now + int64(handlerWrap.ErrorInterval)
							break
						}
					}
				}
			}
		}(i, handlerWrap.ticker[i], handlerWrap.stop[i])
	}
	return nil
}

func (this *queueImpl) tryDoOneTask(lockName, topic, redisKeyQueue, redisKeyHandling string, handlerWrap *topicHandlerWrap) (handled, success bool, err error) {
	n, err := this.client.LLen(redisKeyHandling).Result()
	if err != nil && err != redis.Nil {
		return false, false, err
	} else if n != 0 {
		return this.doOneTask(lockName, topic, redisKeyQueue, redisKeyHandling, handlerWrap)
	}

	data, err := this.client.RPopLPush(redisKeyQueue, redisKeyHandling).Result()
	if err != nil && err != redis.Nil {
		return false, false, err
	} else if data == "" {
		return false, false, nil
	}

	return this.doOneTask(lockName, topic, redisKeyQueue, redisKeyHandling, handlerWrap)
}

func (this *queueImpl) doOneTask(lockName, topic, redisKeyQueue, redisKeyHandling string, handlerWrap *topicHandlerWrap) (handled, success bool, redisError error) {
	// 加锁
	ok, err := this.redisLock.Lock(lockName, handlerWrap.Timeout*2, handlerWrap.Timeout)
	if err != nil {
		return false, false, err
	} else if !ok {
		return false, false, nil
	}
	defer this.redisLock.Unlock(lockName)

	strList, err := this.client.LRange(redisKeyHandling, 0, 0).Result()
	if err != nil && err != redis.Nil {
		return false, false, err
	}

	if len(strList) == 0 {
		return false, false, nil
	}

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("redisTaskQueue.topicHandler[%s] panic: %v", topic, e)
			os.Stderr.WriteString(fmt.Sprintf("[%v] redisTaskQueue.topicHandler[%s] panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), topic, e))
			debug.PrintStack()
		}
	}()

	if this.counter != nil {
		this.counter.Add(1)
	}

	if handlerWrap.handler(topic, strList[0]) {
		this.client.LPop(redisKeyHandling)
		return true, true, nil
	} else {
		return true, false, nil
	}
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
