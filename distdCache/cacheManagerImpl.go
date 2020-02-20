package distdCache

import (
	"fmt"
	"github.com/go-redis/redis"
	"strings"
	"sync"
	"time"
	"yelo/go-util/jsonUtil"
	"yelo/go-util/redisLock"
	"yelo/go-util/taskQueue/chanTaskQueue"
	"yelo/go-util/timeRoundedCounter"
	"yelo/go-util/timeUtil"
)

const (
	msgQueueChannel = "DistdCache:Channel"
)

type cacheManagerImpl struct {
	clientId      string                // 客户端 ID，分布式系统中的每个客户端应该有独立的 ID。
	redisClient   *redis.Client         //
	opt           *CacheManagerOptions  //
	msgQueue      chanTaskQueue.Queue   //
	notifyQueue   chanTaskQueue.Queue   //
	subscriber    *redis.PubSub         // 消息订阅者
	cacheInstance map[string]*cacheImpl //
	checkTicker   timeUtil.Ticker       //
	instanceLock  sync.RWMutex          //
	redisLock     redisLock.RedisLock   //
}

type msgQueueData struct {
	ClientId string `json:"c" description:"发送者客户端ID"`
	Name     string `json:"n" description:"缓存名称"`
	Opr      string `json:"opr" description:"operator, set|del"`
	Key      string `json:"k" description:"key"`
	Val      string `json:"v,omitempty" description:"value"`
	Time     int64  `json:"t" description:"消息的更新时间（ms）"`
}

type notifyQueueData struct {
	f      OnChangeFunc
	opr    string
	key    string
	val    CacheEntity
	source string
}

func (this *cacheManagerImpl) ClientId() string {
	return this.clientId
}

func (this *cacheManagerImpl) ensureStart() error {
	this.instanceLock.Lock()
	defer this.instanceLock.Unlock()

	if this.checkTicker != nil {
		return nil
	}

	// 检查参数
	if this.clientId = strings.TrimSpace(this.clientId); this.clientId == "" {
		return fmt.Errorf("必须设置 clientId")
	}
	if this.redisClient == nil {
		return fmt.Errorf("必须设置 redisClient")
	}

	// 测试 redis
	if err := this.redisClient.Ping().Err(); err != nil {
		return fmt.Errorf("无法访问 redis: %v", err)
	}

	// 初始化其他启动参数
	this.msgQueue = chanTaskQueue.New("distdCacheSubscribe", this.opt.QueueCapicity, func(v interface{}, t time.Time) {
		msg := v.(*msgQueueData)
		this.instanceLock.RLock()
		cache := this.cacheInstance[msg.Name]
		this.instanceLock.RUnlock()
		if cache != nil && cache.started {
			val := &CacheEntity{Time: msg.Time}
			if msg.Opr == Operator_Set {
				val.Data = cache.newEntity(msg.Val)
			}
			cache.doEdit(msg.Opr, msg.Key, val, msg.ClientId)
		}
	}, &chanTaskQueue.Options{
		Counter: timeRoundedCounter.New(5*time.Minute, 60),
	})
	this.notifyQueue = chanTaskQueue.New("distdCacheNotify", this.opt.QueueCapicity, func(v interface{}, t time.Time) {
		msg := v.(*notifyQueueData)
		msg.f(msg.opr, msg.key, msg.val, msg.source)
	}, &chanTaskQueue.Options{
		Counter: timeRoundedCounter.New(5*time.Minute, 60),
	})

	// 启动各个组件
	if err := this.msgQueue.Start(); err != nil {
		return fmt.Errorf("启动消息订阅队列失败: %v", err)
	}
	if err := this.notifyQueue.Start(); err != nil {
		return fmt.Errorf("启动通知队列失败: %v", err)
	}
	if err := this.startMsgSubscriber(); err != nil {
		return fmt.Errorf("启动消费者失败: %v", err)
	}

	this.checkTicker = timeUtil.NewTicker(this.opt.SyncCheckInterval, this.opt.SyncCheckInterval, this.checkSync)

	return nil
}

func (this *cacheManagerImpl) NewCache(name string, newEntityFunc func() interface{}, opt *CacheOption) (Cache, error) {
	// 检查参数
	if name = strings.TrimSpace(name); name == "" {
		return nil, fmt.Errorf("参数 name 不能为空")
	}

	// 检查参数
	var realOpt *CacheOption
	if opt != nil {
		realOpt = opt
	} else {
		realOpt = &CacheOption{}
	}

	if realOpt.BucketCount <= 0 {
		realOpt.BucketCount = 100
	} else if realOpt.BucketCount > 4096 {
		realOpt.BucketCount = 4096
	}

	this.instanceLock.Lock()
	defer this.instanceLock.Unlock()

	key := strings.ToLower(name)
	if tmp := this.cacheInstance[key]; tmp != nil {
		// 已经存在的话，忽略 opt，不能修改参数
		return tmp, nil
	}

	// 初始化其他启动参数
	instance := &cacheImpl{
		manager:         this,
		opt:             realOpt,
		name:            name,
		newEntityFunc:   newEntityFunc,
		bucketKeyPrefix: fmt.Sprintf("DistdCache:%s", strings.Replace(name, ":", "-", -1)),
		syncLockName:    fmt.Sprintf("DistdCache.%s", strings.Replace(name, ":", "-", -1)),
		buckets:         make([]*bucket, realOpt.BucketCount),
	}
	for i := range instance.buckets {
		instance.buckets[i] = &bucket{data: make(map[string]*CacheEntity)}
	}
	this.cacheInstance[name] = instance

	allCache = append(allCache, instance)

	return instance, nil
}

func (this *cacheManagerImpl) checkSync() {
	this.msgQueue.Pause()
	this.instanceLock.RLock()
	defer func() {
		this.instanceLock.RUnlock()
		this.msgQueue.Resume()
	}()
	for _, instance := range this.cacheInstance {
		instance.checkSync()
	}
}

// 开始消费 kafka 消息
func (this *cacheManagerImpl) startMsgSubscriber() error {
	client := redis.NewClient(this.redisClient.Options())
	this.subscriber = client.Subscribe(msgQueueChannel)
	go func() {
		defer this.subscriber.Close()
		for {
			if msg, err := this.subscriber.ReceiveMessage(); err != nil {
				this.opt.Logger.Error("[%v] receive error: %v", this.clientId, err)
			} else {
				this.consumeOneMessage(msg.Payload)
			}
		}
	}()
	return nil
}

// 消费一条消息，返回该消息是否已被处理
func (this *cacheManagerImpl) consumeOneMessage(data string) error {
	msg := &msgQueueData{}
	if err := jsonUtil.UnmarshalFromString(data, msg); err != nil {
		this.opt.Logger.Warn("消息反序列化失败: %v, data=%v", err, data)
		return nil
	}

	if msg.ClientId == this.clientId {
		// 收到了自己发出的消息
		return nil
	}

	if msg.Name == "" || msg.Key == "" || msg.Time == 0 || (msg.Opr != Operator_Set && msg.Opr != Operator_Del) {
		this.opt.Logger.Warn("消息格式不正确, msg=%v", msg)
		return nil
	}

	this.instanceLock.RLock()
	_, exist := this.cacheInstance[msg.Name]
	this.instanceLock.RUnlock()
	if !exist {
		// 不需要当前 cacheManagerImpl 处理（系统中可能存在很多不同类型的缓存数据，他们以 Name 区分）
		return nil
	}

	if ok, _ := this.msgQueue.Add(msg); !ok {
		return fmt.Errorf("写入消费者队列失败")
	}

	return nil
}
