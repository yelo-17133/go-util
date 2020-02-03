package distdCache

import (
	"crypto/md5"
	"fmt"
	"github.com/go-redis/redis"
	"hash/crc32"
	"go-util/convertor"
	"go-util/jsonUtil"
	"go-util/timeUtil"
	"os"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheImpl struct {
	manager         *cacheManagerImpl  //
	opt             *CacheOption       //
	name            string             //
	newEntityFunc   func() interface{} //
	bucketKeyPrefix string             //
	lockNamePrefix  string             //
	syncLockName    string             //
	buckets         []*bucket          //
	started         bool               //
}

type bucket struct {
	data        map[string]*CacheEntity // 数据
	dataTime    int64                   // 最后一次修改数据的时间（毫秒）
	etag        string                  // 数据签名
	etagTime    int64                   // 最后一次修改 etag 的时间（毫秒，使用 syncCheckInterval 取整后）
	lock        sync.RWMutex            //
	hasEtagLock bool                    // 是否获取了更新 etag 的权限，需要在下一个检查周期更新 etag，并且释放锁
}

type serverEtagData struct {
	etag string
	time int64
}

func (this *cacheImpl) Manager() CacheManager {
	return this.manager
}

func (this *cacheImpl) Name() string {
	return this.name
}

func (this *cacheImpl) Size() int {
	n := 0
	for _, bucket := range this.buckets {
		n += len(bucket.data)
	}
	return n
}

func (this *cacheImpl) MemSize() int {
	return -1
}

func (this *cacheImpl) Start() error {
	if !this.started {
		if err := this.manager.ensureStart(); err != nil {
			return err
		}
		this.started = true

		if tmp := this.manager.redisClient.Type(this.bucketKeyPrefix + ":ETag").Val(); tmp != "none" && tmp != "hash" {
			this.manager.redisClient.Del(this.bucketKeyPrefix + ":ETag")
		}

		if err := this.ForceSync(); err != nil {
			this.started = false
			return err
		}
	}
	return nil
}

func (this *cacheImpl) AllKeys() []string {
	keys := make([]string, 0, this.Size()+64)
	for _, bucket := range this.buckets {
		bucket.lock.RLock()
		for k := range bucket.data {
			keys = append(keys, k)
		}
		bucket.lock.RUnlock()
	}
	sort.Strings(keys)
	return keys
}

func (this *cacheImpl) Get(key string) CacheEntity {
	index := this.getBucketIndexByKey(key)
	bucket := this.buckets[index]
	bucket.lock.RLock()
	defer bucket.lock.RUnlock()
	if val, ok := bucket.data[key]; ok && val.Data != nil {
		return *val
	}
	return emptyEntity
}

func (this *cacheImpl) GetAll() map[string]CacheEntity {
	dict := make(map[string]CacheEntity)
	for _, bucket := range this.buckets {
		bucket.lock.RLock()
		for key, val := range bucket.data {
			if val.Data != nil {
				dict[key] = *val
			}
		}
		bucket.lock.RUnlock()
	}
	return dict
}

func (this *cacheImpl) GetData(key string) interface{} {
	if val := this.Get(key); val.Valid() {
		return val.Data
	}
	return nil
}

func (this *cacheImpl) Set(key string, value interface{}) error {
	if value != nil {
		return this.doEdit(Operator_Set, key, &CacheEntity{Data: value, Time: timeUtil.ToMs(time.Now())}, this.manager.clientId)
	} else {
		return this.doEdit(Operator_Del, key, &CacheEntity{Time: timeUtil.ToMs(time.Now())}, this.manager.clientId)
	}
}

func (this *cacheImpl) Del(key string) error {
	return this.doEdit(Operator_Del, key, &CacheEntity{Time: timeUtil.ToMs(time.Now())}, this.manager.clientId)
}

func (this *cacheImpl) ForceSync() error {
	if !this.started {
		return this.Start()
	}

	locked, err := this.manager.redisLock.Lock(this.syncLockName, this.manager.opt.SyncCheckInterval, 3*time.Second)
	if err != nil {
		return err
	} else if locked {
		defer this.manager.redisLock.Unlock(this.syncLockName)
	}

	for i := range this.buckets {
		if err := this.doSyncBucket(i); err != nil {
			return err
		}
	}
	return nil
}

func (this *cacheImpl) Clear() error {
	if !this.started {
		return fmt.Errorf("请先调用 Start 方法启动缓存")
	}

	// 删除 redis 中的 Data
	keys, err := this.manager.redisClient.Keys(this.bucketKeyPrefix + ":Data:*").Result()
	if len(keys) != 0 {
		err = this.manager.redisClient.Del(keys...).Err()
	}
	if err != nil && err != redis.Nil {
		return fmt.Errorf("清空缓存数据失败: %v", err)
	}
	// 删除 redis 中的 ETag
	this.manager.redisClient.Del(this.bucketKeyPrefix + ":ETag")

	// 立即同步
	for i := range this.buckets {
		if err := this.doSyncBucket(i); err != nil {
			this.manager.opt.Logger.Warn("同步数据失败: %v", err)
			break
		}
	}

	return nil
}

func (this *cacheImpl) newEntity(s string) interface{} {
	if this.newEntityFunc == nil {
		return s
	} else {
		v := this.newEntityFunc()
		if s != "" {
			jsonUtil.UnmarshalFromString(s, v)
		}
		return v
	}
}

// 参数:
//   changeTime: 修改数据的时间戳
//   source: 该修改请求是谁触发的：sync 表示是本地同步机制触发的，其他情况表示对应的 ClientId。
//      如果与 Manager.ClientId 相同则表示是通过本地 Set/Del 接口触发的，否则表示是远程节点发送的同步消息触发的。
func (this *cacheImpl) doEdit(opr, key string, val *CacheEntity, source string) error {
	if !this.started {
		return fmt.Errorf("请先调用 Start 方法启动缓存")
	}

	defer func() {
		if e := recover(); e != nil {
			os.Stderr.WriteString(fmt.Sprintf("[%v] distdCache panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e))
			debug.PrintStack()
		}
	}()

	index := this.getBucketIndexByKey(key)
	redisKey := this.getBucketDataKey(index)
	bucket := this.buckets[index]

	// 先读出本地数据，并根据参数做校验和容错
	bucket.lock.RLock()
	localVal := bucket.data[key]
	bucket.lock.RUnlock()
	if localVal != nil {
		// 如果是通过订阅消息触发的，通过比较 Time 过滤掉旧版本的消息
		if source != this.manager.clientId && source != "sync" && val.Time < localVal.Time {
			return nil
		}
	}

	// 如果是通过调用 Set/Del 接口触发的，将数据写入 Redis 并发送广播消息
	if source == this.manager.clientId {
		// 写入 redis。
		var err error
		if opr == Operator_Set {
			err = this.manager.redisClient.HSet(redisKey, key, jsonUtil.MustMarshalToString(val)).Err()
		} else {
			err = this.manager.redisClient.HDel(redisKey, key).Err()
		}
		if err != nil && err != redis.Nil {
			return fmt.Errorf("写入 redis 失败: %v", err)
		}
		if this.opt.Expire > 0 {
			this.manager.redisClient.Expire(redisKey, this.opt.Expire)
		}

		// 发布消息
		err = this.manager.redisClient.Publish(msgQueueChannel, jsonUtil.MustMarshalToString(&msgQueueData{
			ClientId: this.manager.clientId,
			Name:     this.name,
			Opr:      opr,
			Key:      key,
			Val:      convertor.MustToString(val.Data),
			Time:     val.Time,
		})).Err()
		if err != nil {
			// 此处只记日志但不返回，因为前面写 Redis 如果没有出错，那么此处极大概率此处也不会出错，况且即使出错也不需要特别处理，ETag 同步机制可自动纠正
			this.manager.opt.Logger.Warn("publish msg error: %v", err)
		}
	}

	var changed bool
	var changeData interface{}

	// 第三步：更新本地数据
	bucket.lock.Lock()
	if opr == Operator_Set {
		if localVal == nil {
			changed, changeData = true, val.Data
			bucket.data[key] = &CacheEntity{Data: val.Data, Time: val.Time}
		} else if !reflect.DeepEqual(localVal.Data, val.Data) {
			changed, changeData = true, val.Data
			localVal.Data, localVal.Time = val.Data, val.Time
		}
	} else {
		if localVal != nil && localVal.Data != nil {
			changed, changeData = true, localVal.Data
			localVal.Data, localVal.Time = nil, val.Time
		}
	}
	bucket.lock.Unlock()

	// fire event
	if changed && this.opt.OnChange != nil {
		this.manager.notifyQueue.Add(&notifyQueueData{
			f:      this.opt.OnChange,
			opr:    opr,
			key:    key,
			val:    CacheEntity{Data: changeData, Time: val.Time},
			source: source,
		})
	}

	return nil
}

func (this *cacheImpl) getBucketIndexByKey(key string) int {
	if this.opt.KeyCodeFunc != nil {
		return this.opt.KeyCodeFunc(key) % len(this.buckets)
	} else {
		return int(crc32.ChecksumIEEE([]byte(key)) % uint32(len(this.buckets)))
	}
}

func (this *cacheImpl) getBucketDataKey(index int) string {
	return fmt.Sprintf("%s:Data:%02d", this.bucketKeyPrefix, index)
}

func (this *cacheImpl) getETagLockName(index int) string {
	return fmt.Sprintf("%s.ETag-%02d", this.lockNamePrefix, index)
}

func (this *cacheImpl) checkSync() error {
	locked, err := this.manager.redisLock.Lock(this.syncLockName, this.manager.opt.SyncCheckInterval, 3*time.Second)
	if err != nil {
		return err
	} else if !locked {
		this.manager.opt.Logger.Debug("[%v-%v] checkSync lock faild: %v", this.name, this.manager.clientId, this.syncLockName)
		return fmt.Errorf("获取锁失败")
	}

	defer func() {
		this.manager.redisLock.Unlock(this.syncLockName)
		if e := recover(); e != nil {
			this.manager.opt.Logger.Error("[%v] distdCache.checkSync panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e)
			os.Stderr.WriteString(fmt.Sprintf("[%v] distdCache.checkSync panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e))
			debug.PrintStack()
		}
	}()

	serverETagMap, err := this.getServerEtags()
	if err != nil {
		return fmt.Errorf("获取 ETag 失败: %v", err)
	}

	for i, bucket := range this.buckets {
		bucket.lock.RLock()
		this.updateEtag(bucket, timeUtil.ToMs(time.Now()))
		bucket.lock.RUnlock()
		if serverEtag := serverETagMap[i]; serverEtag == nil && bucket.etag == "" {
			continue
		} else if serverEtag != nil && serverEtag.etag == bucket.etag {
			continue
		} else if err := this.doSyncBucket(i); err != nil {
			// doSyncBucket 只有在 redis 无法访问时会返回错误
			this.manager.opt.Logger.Error("同步数据出错: %v", err)
			return fmt.Errorf("同步数据出错: %v", err)
		}
	}

	return nil
}

// 同步一个 bucket
// 参数:
//   getETagLock: 是否尝试独占向 redis 写入 ETag 的权限（一个同步周期），每次
// 返回值:
//   如果获取 Redis 数据失败，则返回对应的 error。
func (this *cacheImpl) doSyncBucket(index int) error {
	nowMs := timeUtil.ToMs(time.Now())
	bucket := this.buckets[index]

	// 加锁期间，消息通知、Set/Del 接口调用都会被阻塞，直到该 bucket 完成同步
	// 一开始就加锁然后再读取 redis，保守策略，牺牲一部分性能确保数据一致性
	bucket.lock.Lock()
	defer bucket.lock.Unlock()

	serverData, err := this.manager.redisClient.HGetAll(this.getBucketDataKey(index)).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("读取 redis 数据失败: %v", err)
	}

	// 检查已删除（本地存在、但在 redis 中已不存在）的 key
	for key, localVal := range bucket.data {
		if _, ok := serverData[key]; !ok && localVal.Data != nil {
			// update
			localVal.Data, localVal.Time = nil, nowMs
			// fire event
			if this.opt.OnChange != nil {
				this.manager.notifyQueue.Add(&notifyQueueData{
					f:      this.opt.OnChange,
					opr:    Operator_Del,
					key:    key,
					val:    *localVal,
					source: "sync",
				})
			}
		}
	}
	// 加载服务端的 key-value
	for key, valStr := range serverData {
		val := &CacheEntity{}
		if err := jsonUtil.UnmarshalFromString(valStr, val); err != nil {
			continue
		} else if val.Data != nil {
			val.Data = this.newEntity(convertor.MustToString(val.Data))
		}
		localVal, ok := bucket.data[key]
		if !ok || !reflect.DeepEqual(localVal.Data, val.Data) {
			// update
			if localVal == nil {
				bucket.data[key] = val
			} else {
				localVal.Data, localVal.Time = val.Data, val.Time
			}

			// fire event
			if this.opt.OnChange != nil {
				var opr string
				if val.Data != nil {
					opr = Operator_Set
				} else {
					opr = Operator_Del
				}
				this.manager.notifyQueue.Add(&notifyQueueData{
					f:      this.opt.OnChange,
					opr:    opr,
					key:    key,
					val:    *val,
					source: "sync",
				})
			}
		}
	}

	// 重新计算 ETag，并写入服务端
	this.updateEtag(bucket, nowMs)
	etagRedisKey := this.bucketKeyPrefix + ":ETag"
	if bucket.etag == "" {
		this.manager.redisClient.HDel(etagRedisKey, strconv.Itoa(index))
	} else {
		this.manager.redisClient.HSet(etagRedisKey, strconv.Itoa(index), fmt.Sprintf("%v-%v", bucket.etag, bucket.etagTime))
	}

	return nil
}

func (this *cacheImpl) getServerEtags() (map[int]*serverEtagData, error) {
	etagRedisKey := this.bucketKeyPrefix + ":ETag"

	// 获取服务端的所有 ETag
	dict, err := this.manager.redisClient.HGetAll(etagRedisKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	var keysToDel []string
	serverETagMap := make(map[int]*serverEtagData, len(this.buckets))
	for key, val := range dict {
		if index, err := strconv.ParseInt(key, 10, 64); err != nil || index < 0 || int(index) >= this.opt.BucketCount {
			keysToDel = append(keysToDel, key)
		} else if segs := strings.Split(val, "-"); len(segs) != 2 {
			keysToDel = append(keysToDel, key)
		} else if version, err := strconv.ParseInt(segs[1], 10, 64); err != nil {
			keysToDel = append(keysToDel, key)
		} else {
			serverETagMap[int(index)] = &serverEtagData{etag: segs[0], time: version}
		}
	}
	if len(keysToDel) != 0 {
		this.manager.redisClient.HDel(etagRedisKey, keysToDel...)
	}

	return serverETagMap, nil
}

func (this *cacheImpl) updateEtag(bucket *bucket, nowMs int64) bool {
	/**
	!!! 注意：此函数内部不要对 bucket 加锁，调用者已经加锁。里面再加锁就会死锁
	*/
	interval := int64(this.manager.opt.SyncCheckInterval / time.Millisecond)
	if etagTime := (nowMs / interval) * interval; etagTime != bucket.etagTime {
		delIfTimeBefore := nowMs - interval*2
		arr, keysToDel := make([]string, 0, len(bucket.data)), make([]string, 0, 64)
		for k, v := range bucket.data {
			if v.Time > etagTime {
				// 计算 ETag 时会按 SyncCheckInterval 取整，计算在整点之前的数据对应的 ETag。所以 >etagTime 的忽略不参与 Etag 计算
				continue
			} else if v.Data != nil {
				arr = append(arr, k+"="+convertor.MustToString(v))
			} else if v.Time < delIfTimeBefore {
				keysToDel = append(keysToDel, k)
			}
		}
		for _, k := range keysToDel {
			delete(bucket.data, k)
		}
		var etag string
		if len(arr) != 0 {
			sort.Strings(arr)
			etag = fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(arr, "\n"))))
		}
		if etag != bucket.etag {
			bucket.etag, bucket.etagTime = etag, etagTime
			return true
		}
	}
	return false
}
