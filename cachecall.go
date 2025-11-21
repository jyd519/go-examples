package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// CachedResult 存储缓存的结果和其过期时间
type CachedResult[T any] struct {
	Value      T
	Error      error
	Expiration time.Time
}

// inFlightRequest 表示正在处理中的请求
type inFlightRequest struct {
	done chan struct{} // 信号请求完成
}

// FunctionCache 提供缓存功能的结构体
type FunctionCache struct {
	mu       sync.RWMutex
	cache    map[string]interface{}
	inFlight map[string]*inFlightRequest // 跟踪正在处理中的请求
}

// NewFunctionCache 创建一个新的函数缓存器
func NewFunctionCache() *FunctionCache {
	return &FunctionCache{
		cache:    make(map[string]interface{}),
		inFlight: make(map[string]*inFlightRequest),
	}
}

// WithCache 执行函数并缓存结果5分钟
// 如果缓存中已有有效结果，则直接返回缓存的结果
// 如果有另一个线程正在计算相同的结果，则等待该线程完成并使用其结果
func WithCache[T any](fc *FunctionCache, cacheKey string, fn func() (T, error)) (T, error) {
	// 首先检查是否有有效的缓存
	fc.mu.RLock()
	if cached, ok := fc.cache[cacheKey]; ok {
		if typedCache, ok := cached.(*CachedResult[T]); ok {
			if time.Now().Before(typedCache.Expiration) {
				fc.mu.RUnlock()
				return typedCache.Value, typedCache.Error
			}
		}
	}

	// 检查是否已有相同的请求正在处理
	if req, ok := fc.inFlight[cacheKey]; ok {
		// 有请求正在处理，释放读锁并等待
		fc.mu.RUnlock()
		<-req.done // 等待已有的请求完成

		// 请求完成后，再次检查缓存
		fc.mu.RLock()
		if cached, ok := fc.cache[cacheKey]; ok {
			if typedCache, ok := cached.(*CachedResult[T]); ok {
				fc.mu.RUnlock()
				return typedCache.Value, typedCache.Error
			}
		}
		fc.mu.RUnlock()

		// 如果走到这里，意味着有问题（应该不会发生）
		// 为安全起见，执行函数并返回结果但不缓存
		return fn()
	}

	// 没有请求正在处理，准备执行函数
	req := &inFlightRequest{done: make(chan struct{})}
	fc.mu.RUnlock()

	// 升级到写锁，再次检查状态（防止在获取写锁期间状态改变）
	fc.mu.Lock()

	// 双重检查缓存
	if cached, ok := fc.cache[cacheKey]; ok {
		if typedCache, ok := cached.(*CachedResult[T]); ok {
			if time.Now().Before(typedCache.Expiration) {
				fc.mu.Unlock()
				return typedCache.Value, typedCache.Error
			}
		}
	}

	// 双重检查是否有请求正在处理
	if existingReq, inFlightExists := fc.inFlight[cacheKey]; inFlightExists {
		fc.mu.Unlock()
		<-existingReq.done // 等待已有的请求完成

		// 请求完成后，获取缓存结果
		fc.mu.RLock()
		if cached, ok := fc.cache[cacheKey]; ok {
			if typedCache, ok := cached.(*CachedResult[T]); ok {
				fc.mu.RUnlock()
				return typedCache.Value, typedCache.Error
			}
		}
		fc.mu.RUnlock()

		// 安全起见
		return fn()
	}

	// 标记该键有请求正在处理
	fc.inFlight[cacheKey] = req
	fc.mu.Unlock()

	// 确保在函数返回时发出完成信号并清理 inFlight
	defer func() {
		fc.mu.Lock()
		close(req.done)
		delete(fc.inFlight, cacheKey)
		fc.mu.Unlock()
	}()

	// 执行函数获取结果
	result, err := fn()

	// 缓存结果
	cached := &CachedResult[T]{
		Value:      result,
		Error:      err,
		Expiration: time.Now().Add(5 * time.Minute),
	}

	fc.mu.Lock()
	fc.cache[cacheKey] = cached
	fc.mu.Unlock()

	return result, err
}

// ClearCache 清除指定键的缓存
func (fc *FunctionCache) ClearCache(cacheKey string) {
	fc.mu.Lock()
	delete(fc.cache, cacheKey)
	fc.mu.Unlock()
}

// ClearAllCache 清除所有缓存
func (fc *FunctionCache) ClearAllCache() {
	fc.mu.Lock()
	fc.cache = make(map[string]interface{})
	fc.mu.Unlock()
}

// 模拟一个耗时的数据库查询函数
func expensiveDBQuery(id int) (string, error) {
	log.Printf("[%v] Executing expensive DB query for ID: %d", time.Now().Format("15:04:05.000"), id)
	// 模拟查询延迟
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Result for ID %d", id), nil
}

func main() {
	// 创建一个缓存实例
	fc := NewFunctionCache()

	// 测试并发调用
	var wg sync.WaitGroup

	// 启动多个并发请求
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()

			log.Printf("[%v] Goroutine %d requesting data", time.Now().Format("15:04:05.000"), num)
			start := time.Now()

			result, err := WithCache(fc, "db-query-1", func() (string, error) {
				return expensiveDBQuery(1)
			})

			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[%v] Goroutine %d error: %v", time.Now().Format("15:04:05.000"), num, err)
			} else {
				log.Printf("[%v] Goroutine %d result: %s (took %v)",
					time.Now().Format("15:04:05.000"), num, result, elapsed)
			}
		}(i)

		// 稍微错开请求时间，使并发效果更明显
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	// 等待缓存过期，然后再次测试
	log.Printf("\n[%v] Waiting for cache to expire...", time.Now().Format("15:04:05.000"))
	// 为了演示，我们可以手动清除缓存，而不是等待5分钟
	time.Sleep(1 * time.Second)
	fc.ClearCache("db-query-1")

	// 再次进行并发测试
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()

			log.Printf("[%v] Goroutine %d requesting data after cache cleared",
				time.Now().Format("15:04:05.000"), num)
			start := time.Now()

			result, _ := WithCache(fc, "db-query-1", func() (string, error) {
				return expensiveDBQuery(2)
			})

			elapsed := time.Since(start)
			log.Printf("[%v] Goroutine %d result: %s (took %v)",
				time.Now().Format("15:04:05.000"), num, result, elapsed)
		}(i)

		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()
}
