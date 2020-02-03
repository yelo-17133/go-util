// ------------------------------------------------------------------------------
// 利用 Redis 实现的分布式锁。
//
// 在多线程开发时使用锁时，需要通过 try-finally 来确保锁的释放，否则如果代码执行到一半出现异常导致锁没有机会被释放，就有可能导致死锁。
// 但即使没有使用 try-finally，在发现有问题时也可以通过重启进程来解决，因为操作系统会在进程退出时释放这些资源。所以，多线程的死锁漏洞是可以通过进程重启解决的。
//
// 但在使用分布式程序时，是没有这样的机制能够通过重启进程或者重启服务器来自动释放分布式锁的，所以分布式锁的释放必须由业务逻辑来保证锁一定会被释放、不会出现死锁。
// 我们使用 Redis 来实现分布式锁，通过使用 String.SetNx 来实现加锁，通过 String.Del 来主动释放锁，通过 TTL 来确保锁一定会被释放：
// 当通过 SetNx 获得锁时，同时会给锁设置一个 TTL，该 TTL 表示这个锁如果超过这个时间仍未被释放，则由 Redis 负责释放。因此该 TTL 应该满足：
//   1、该 TTL 应该足够大：所有的任务都一定会在 TTL 之内被处理掉，不会由于某个任务处理时间过长而导致锁被自动释放、从而让另一个消费线程拿到了锁、导致有两个消费线程都进入互斥区。
//   2、该 TTL 应该足够小：如果某个消费线程由于进程奔溃或者服务器掉电等原因导致未被释放，则在 TTL 之后锁会被 Redis 自动释放，从而让其他的消费线程有机会重新获得锁，不至于被挂起太长时间从而影响并发。
// 因此，有以下建议：
// 如果互斥区的代码执行时间比较短，那么可以设置为 加锁后代码要执行的最大耗时 * N(=2)。
// 如果互斥区的代码执行时间比较长（比如需要一个 for 循环做很多事情），那么可以给 TTL 设置一个合理的时间，并在代码执行过程中不断的刷新 TTL。如：
// if Lock(xxx, 5秒) {
//    nextRefreshTtl = time.Now().Add(2.5秒)
//    for i := 0; i : 100; i ++ {
//       do something;
//       if time.Now().After(nextRefreshTtl) {
//          Expire(xxx, 5秒)
//          nextRefreshTtl = nextRefreshTtl.Add(2.5秒)
//       }
//    }
// }
// 特别注意：由于刷新 TTL 是有耗时的，计算机的时间也会有偏差，所以一定要在 TTL 到期之前的足够时间内完成刷新，以免出现 Redis 已经由于 TTL 到期把锁给释放了之后才收到客户端的刷新请求。
// ------------------------------------------------------------------------------
package redisLock

import (
	"fmt"
	"github.com/go-redis/redis"
	"strings"
	"time"
)

type RedisLock interface {
	// 获取 Redis 连接对象
	RedisClient() *redis.Client
	// 设置 Redis 连接对象
	SetRedisClient(*redis.Client)
	// 获取创建对象时设置的分组名，该分组名用来防止不同模块的锁 key 重复。
	Group() string
	// 加锁。如果在超时时间内获得了锁，则返回 true，否则返回 false。 timeout<=0 表示加锁失败时立即返回、不等待。
	Lock(key string, ttl, timeout time.Duration) (bool, error)
	// 释放锁
	Unlock(key string) error
	// 延长锁的失效时间
	Expire(key string, ttl time.Duration) error
}

type Options struct {
	WaitTimeout time.Duration // 加锁时如果锁已经被占用，默认最长等待时间。
	Ttl         time.Duration // 锁的有效期，超过有效期仍未主动释放的话则由系统自动释放。最小 3 秒。
}

func New(client *redis.Client, group string) RedisLock {
	if group = strings.TrimSpace(group); group == "" {
		group = "Default_"
	}
	return &redisLock{client: client, group: group}
}

const (
	minTtl       = 3 * time.Second
	waitInterval = 10 * time.Millisecond
)

func (this *Options) update() *Options {
	if this.Ttl < minTtl {
		this.Ttl = minTtl
	}
	return this
}

type redisLock struct {
	client *redis.Client // redis 连接
	group  string        // 分组名，防止不同模块的锁 key 重复
}

// 获取 Redis 连接对象
func (this *redisLock) RedisClient() *redis.Client {
	return this.client
}

// 设置 Redis 连接对象
func (this *redisLock) SetRedisClient(client *redis.Client) {
	this.client = client
}

// 获取创建对象时设置的分组名，该分组名用来防止不同模块的锁 key 重复。
func (this *redisLock) Group() string {
	return this.group
}

// 加锁。如果在超时时间内获得了锁，则返回 true，否则返回 false。 timeout<=0 表示加锁失败时立即返回、不等待。
func (this *redisLock) Lock(key string, ttl, timeout time.Duration) (bool, error) {
	if ttl < minTtl {
		ttl = minTtl
	}
	if key = strings.TrimSpace(strings.Replace(key, ":", "-", -1)); key == "" {
		key = "default"
	}

	redisKey := fmt.Sprintf("RedisLock:%s:%s", this.group, key)
	ok, err := this.client.SetNX(redisKey, "", ttl).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}

	if ok {
		return true, nil
	}

	if timeout > 0 {
		timer, timeout := time.NewTicker(waitInterval), time.Now().Add(timeout)
		for range timer.C {
			if this.client.SetNX(redisKey, "", ttl).Val() {
				timer.Stop()
				return true, nil
			} else if time.Now().After(timeout) {
				timer.Stop()
				return false, nil
			}
		}
	}

	return false, nil
}

// 释放锁
func (this *redisLock) Unlock(key string) error {
	if key = strings.TrimSpace(strings.Replace(key, ":", "-", -1)); key == "" {
		key = "default"
	}

	redisKey := fmt.Sprintf("RedisLock:%s:%s", this.group, key)
	if err := this.client.Del(redisKey).Err(); err != nil && err != redis.Nil {
		return err
	}

	return nil
}

// 延长锁的失效时间
func (this *redisLock) Expire(key string, ttl time.Duration) error {
	if ttl < minTtl {
		ttl = minTtl
	}
	if key = strings.TrimSpace(strings.Replace(key, ":", "-", -1)); key == "" {
		key = "default"
	}

	redisKey := fmt.Sprintf("RedisLock:%s:%s", this.group, key)
	if err := this.client.Expire(redisKey, ttl).Err(); err != nil && err != redis.Nil {
		return err
	}

	return nil
}
