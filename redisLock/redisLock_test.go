package redisLock

import (
	"fmt"
	"github.com/go-redis/redis"
	"go-util/_utilTest"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_utilTest.Init()
	m.Run()
}

func TestRedisLockUnlock(t *testing.T) {
	lock := New(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), "test").(*redisLock)
	var ok bool
	var err error

	for i := 0; i < 10; i++ {
		ok, err = lock.Lock("abc", 5*time.Second, 0)
		if err != nil {
			t.Errorf("error occured: %v", err)
			return
		}
		if !ok {
			t.Errorf("assert faild")
			return
		}
		err = lock.Unlock("abc")
		if err != nil {
			t.Errorf("error occured: %v", err)
			return
		}
	}
}

func TestRedisLockRelock(t *testing.T) {
	lock := New(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), "test").(*redisLock)
	var ok bool
	var err error
	var begin time.Time
	var timespan time.Duration

	ok, err = lock.Lock("abc", 5*time.Second, 0)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if !ok {
		t.Errorf("assert faild")
		return
	}
	begin = time.Now()
	ok, err = lock.Lock("abc", 5*time.Second, 0)
	timespan = time.Now().Sub(begin)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if ok {
		t.Errorf("assert faild")
		return
	}
	if timespan > 50*time.Millisecond {
		t.Errorf("assert faild: %v", timespan)
		return
	}
}

func TestRedisLockTimeout(t *testing.T) {
	lock := New(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), "test").(*redisLock)
	var ok bool
	var err error
	var begin time.Time
	var timespan time.Duration

	err = lock.Unlock("abc")
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	ok, err = lock.Lock("abc", 5*time.Second, 0)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	begin = time.Now()
	ok, err = lock.Lock("abc", 5*time.Second, 3*time.Second)
	timespan = time.Now().Sub(begin)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if ok {
		t.Errorf("assert faild")
		return
	}
	if timespan < 3*time.Second {
		t.Errorf("assert faild: %v", timespan)
		return
	}
	t.Log(fmt.Sprintf("timespan: %v", timespan))
}

func TestRedisLockExpired(t *testing.T) {
	lock := New(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), "test").(*redisLock)
	var ok bool
	var err error
	var begin time.Time
	var timespan time.Duration

	err = lock.Unlock("abc")
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	ok, err = lock.Lock("abc", 5*time.Second, 0)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	begin = time.Now()
	ok, err = lock.Lock("abc", 5*time.Second, 6*time.Second)
	timespan = time.Now().Sub(begin)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if !ok {
		t.Errorf("assert faild")
		return
	}
	if timespan < 5*time.Second {
		t.Errorf("assert faild: %v", timespan)
		return
	}
	t.Log(fmt.Sprintf("timespan: %v", timespan))
}

func TestRedisLockExpire(t *testing.T) {
	lock := New(redis.NewClient(&redis.Options{Addr: _utilTest.RedisAddr, Password: _utilTest.RedisPassword}), "test").(*redisLock)
	var ok bool
	var err error
	var timespan time.Duration

	err = lock.Unlock("abc")
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}

	ttl, expire, expireInterval := 3*time.Second, 3, 1*time.Second

	ok, err = lock.Lock("abc", ttl, 0)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	go func() {
		for i := 0; i < expire; i++ {
			lock.Expire("abc", ttl)
			time.Sleep(expireInterval)
		}
	}()

	begin, timeout := time.Now(), 5*time.Second
	ok, err = lock.Lock("abc", ttl, timeout)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if ok {
		t.Errorf("assert faild")
		return
	}
	ok, err = lock.Lock("abc", ttl, time.Hour)
	timespan = time.Now().Sub(begin)
	if err != nil {
		t.Errorf("error occured: %v", err)
		return
	}
	if !ok {
		t.Errorf("assert faild")
		return
	}
	if timespan < time.Duration(expire-1)*expireInterval+ttl {
		t.Errorf("assert faild: %v", timespan)
		return
	}
	t.Log(fmt.Sprintf("timespan: %v", timespan))
}
