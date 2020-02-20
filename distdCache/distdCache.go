// ------------------------------------------------------------------------------
// 分布式内存缓存(distributed memory cache)
// 在分布式系统中，如果多个节点都会对同一份缓存数据做写操作，则不同节点之间的缓存数据就容易出现不一致。
// 但如果使用redis集中式缓存，当缓存数据的访问频率很高的时候，就会对redis造成很大的压力。
// 有时候我们需要在分布式系统中维持一份一致的内存缓存，对于这份数据我们希望：
//   1、数据缓存在内存中，这样读取数据时从本地内存中读取，提升读取速度。
//   2、读取频率高，或者在某些情况下会短时间内产生大量读取数据的请求。
//   3、在分布式系统中，如果其中一个节点如果修改了缓存数据，则其他节点应当一起修改，保持所有节点之间的数据完全一致。
//   4、不同各节点之间对数据一致性的要求并不是非常严格，数据的同步间隔在百毫秒级别是可以接受的（大多数分布式消息队列的同步间隔都是在 10 or 100 毫秒级别）
//   4、总数据量小（低于100M），即使全量同步一份数据也不需要耗费过长时间。
//
// 本模块用来实现类似的缓存组件。实现方案如下：
// 以 redis 作为标准存储，用来保存最新版的数据，当节点间的数据存在不一致时以 redis 为准。
// 以 redis 的消息订阅和分发作为消息队列，实现数据变更通知，当某个节点修改了缓存数据时，通知其他节点。
// 保险起见，为防止在消息发布和订阅过程中出现异常导致消息丢失，模块增加定期数据检查与同步机制，即使在这种极端情况下也能正确同步数据。
//
//
// 具体方案：
// 模块在内存中维持 key-value 数据缓存副本，同时附加保存数据的最后修改时间。
// 每次修改数据前，先向 redis 写入数据，确保 redis 中的数据是最准确的并作为同步基准。同时通过消息队列通知其他节点数据已经发生变更。
// 节点监听消息队列，当收到数据变更通知时，根据消息内容修改本地缓存数据。 考虑到消息存在延迟，所以在收到消息时先对比消息和本地的数据更新时间，判断哪边的数据更新。
//   注意!!! 这里假定且依赖系统中的各节点时间是同步的
//   由于需要对比修改时间，所以对于del操作，不能直接删除，而是先通过一个字段deleted标记该key是否被删除，延迟一段时间（大致为 2 个同步间隔）再删除
// 各模块定期比较本地数据与 redis 中数据的一致性，如果发现本地与 redis 有差异则从服务端同步最新数据。
//
// 数据同步的效率：
// 假设大多数情况下本地数据与远程数据是一致的（而且确实是这样）。为了减少比较时的数据传输量，通过比较数据集合的签名来判断数据是否一致、而不是逐个key-value比较。
// 当本地签名与服务端签名不一致时就需要同步数据，为了减少数据不一致时需同步的数据量，把数据分为多个bucket，每个bucket计算一个签名，这样当个别key数据不一致时，只需要同步对应的bucket即可。
//
// 服务端数据签名的维护：
// 签名都是在各个客户端计算的，redis本身并不支持对某个hash计算签名。所以服务端的签名也是依赖于客户端写入的。
// 分布式系统中如果不同节点之间出现数据不一致，则必然出现签名不一致，那么由谁来负责向服务端写入签名、如何正确的写入签名就非常重要。
// 由于各节点之间的数据同步有耗时，所以每次比较签名时不能简单的用本地最新的签名与服务端最新签名比较，这样的话一旦有尚在队列中的消息就会匹配失败、从而触发不必要的数据同步。因此：
//   只记录和同步时间间隔整数倍时间点时各个节点的签名状态，这样当某个节点还没消费消息时，它上一个时间点的签名和其它节点也是匹配的，不会触发同步。
//   同时每个节点在做完检查后，需要删除过期的签名，只保留距离检查时间最近的一个时间点的签名。
//   这样当某个节点真的存在通过同步数据失败的情况，它下一个时间点的签名就是错误的，会在下一次检查时出现签名不匹配的情况，从而触发同步，获得正确数据。
// 为了保持节点在每个时间点的签名状态：
//   当节点调用set和del操作，或者收到消息需要改变本地数据时，会检查数据修改时间是否到达了下一个时间点，若到达了，则先更新签名，再更新本地数据。
//   在每个节点在定时检查时，若发现自身的签名时间点落后于当前时间，则更新签名。
//
// 本地与服务端签名对比流程：
// 每个节点都启动一个间隔T(=1)分钟的定时器，用本地N个Bucket生成的N个ETag，与服务端的比较。如果没有匹配的ETag，说明这个bucket的数据出现了不一致，需要同步。
// 服务端的签名，正常情况下由第一个进入最新时间点的节点，作为主节点负责写入到redis的hash中，哈希中的key为bucketId，value为"{BucketETag}-{ETagLastTime}"。
// 一开始服务端的是签名为空，此时第一个进入检查的节点需要从redis中同步bucket的数据，然后更新签名。
// 如果有节点发现签名不匹配，则从redis上同步数据，此时其数据必然是最新的，在根据规则更新准点签名后，可立即写入redis。
// 即使此时该节点中的部分数据是不正确的、且导致写入了错误的签名，但此后其他节点在比较签名时就会发现签名不匹配，重复执行上一步即可将签名修复，从而使该节点在下次比较签名时又触发数据同步，从而纠正错误数据。
//
// 效果：
// 综上，分布式内存缓存在正常情况下各节点数据都是保持一致的。
// 即使由于特殊原因导致数据不一致（包括但不限于：1、主动修改redis中的缓存数据；2、节点修改了redis数据后产生panic导致消息未能发出去；3、节点在收到消息后回调函数产生panic导致未能正确处理消息），最多经过3个同步周期，即可将数据修复。
// ------------------------------------------------------------------------------
package distdCache

import (
	"github.com/go-redis/redis"
	"strings"
	"time"
	"yelo/go-util/log"
	"yelo/go-util/redisLock"
	"yelo/go-util/strUtil"
	"yelo/go-util/timeUtil"
)

const (
	Operator_Set = "set"
	Operator_Del = "del"
)

type CacheManager interface {
	// 获取 ClientId
	ClientId() string
	// 获取一个 Cache 实例
	NewCache(name string, newEntityFunc func() interface{}, opt *CacheOption) (Cache, error)
}

type Cache interface {
	//
	Manager() CacheManager
	//
	Name() string
	// 获取缓存数量
	Size() int
	//
	MemSize() int
	// 启动缓存（并从服务端同步数据）。启动之前本地数据为空，任何操作都会失败
	Start() error
	// 获取所有的 Key
	AllKeys() []string
	// 获取一个值，key 区分大小写
	Get(key string) CacheEntity
	// 获取所有的值
	GetAll() map[string]CacheEntity
	// 获取一个值，key 区分大小写
	GetData(key string) interface{}
	// 设置一个值，key 区分大小写
	Set(key string, val interface{}) error
	// 删除一个值，key 区分大小写
	Del(key string) error
	// 清空数据
	Clear() error
	// 强制从服务端同步数据。
	// 一般情况下不需要做此操作，因为同步机制会定期对比本地数据与服务端数据，发现不一致时会自动同步。
	ForceSync() error
}

type CacheManagerOptions struct {
	// 检查数据一致性的间隔，默认 1 分钟
	SyncCheckInterval time.Duration
	// （用于通知数据变更的）消息队列最大容量，默认 102400
	QueueCapicity int
	// 记录器
	Logger log.Logger
}

type CacheOption struct {
	BucketCount int                  // 桶的数量，默认 100
	Expire      time.Duration        // 过期时间
	OnChange    OnChangeFunc         // 当数据发生改变时要执行的回调函数
	KeyCodeFunc func(key string) int // 获取 Key 对应的 hashCode 的函数。数据将会放在 hashCode % BucketCount 对应的桶中。默认为 Crc32IEEE。
}

// 当缓存数据发生改变时的事件回调函数
//   opr: set|del
//   key: 发生改变的 key
//   val: 缓存数据
//   source: 事件来源，clientId 或者 sync（表示是同步逻辑触发的）
type OnChangeFunc func(opr, key string, val CacheEntity, source string)

type CacheEntity struct {
	Data interface{} `json:"data,omitempty" description:"缓存的值，如果构造缓存实例时指定了 newEntityFunc ，则为该函数返回的实例；否则为字符串"`
	Time int64       `json:"time,omitempty" description:"最后一次修改的时间（毫秒）"`
}

func (this CacheEntity) Valid() bool {
	return this.Time > timeUtil.Local2000Ms
}

var (
	allCache    = make([]Cache, 0)
	emptyEntity = CacheEntity{}

	DefaultCacheManagerOptions = CacheManagerOptions{}
)

// 创建一个 CacheManager 实例
//   clientId: 客户端唯一ID。在分布式系统中，请确保不同实例的 ClientId 不同，否则具有相同 ClientId 的实例会出现数据不完整的情况。
//   redisClient: redis 连接池
//   opt: 其他参数
func NewCacheManager(clientId string, redisOpt *redis.Options, opt *CacheManagerOptions) CacheManager {
	// ensure clientId
	if clientId = strings.TrimSpace(clientId); clientId == "" {
		clientId = strUtil.Rand(4)
	}

	// ensure realOpt
	var realOpt *CacheManagerOptions
	if opt != nil {
		realOpt = opt
	} else {
		realOpt = &CacheManagerOptions{}
		*realOpt = DefaultCacheManagerOptions
	}

	// ensure realOpt.***
	if realOpt.SyncCheckInterval <= 0 {
		realOpt.SyncCheckInterval = time.Minute
	}
	if realOpt.QueueCapicity <= 0 {
		realOpt.QueueCapicity = 10240
	}
	if realOpt.Logger == nil {
		realOpt.Logger = log.EmptyLogger()
	}

	redisClient := redis.NewClient(redisOpt)
	return &cacheManagerImpl{
		clientId:      clientId,
		redisClient:   redisClient,
		cacheInstance: make(map[string]*cacheImpl, 16),
		redisLock:     redisLock.New(redisClient, "DistdCache"),
		opt:           realOpt,
	}
}

func AllCache() []Cache {
	return allCache
}
