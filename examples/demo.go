/**
 * @Author: guxline zjguoxin@163.com
 * @Date: 2025/7/3 05:39:06
 * @LastEditors: guxline zjguoxin@163.com
 * @LastEditTime: 2025/7/3 05:39:06
 * Description:
 * Copyright: Copyright (©) 2025 中易综服. All rights reserved.
 */
package main

import (
	"fmt"
	"time"

	"github.com/zjguoxin/goscache/cache"
)

func main() {
	// ========== 1. 初始化缓存 ==========
	fmt.Println("=== 初始化缓存 ===")

	// 方式1：内存缓存（默认配置）
	memCache, err := cache.NewCache(cache.CacheTypeMemory)
	if err != nil {
		panic(err)
	}
	defer memCache.Close()

	// 方式2：Redis缓存（自定义配置）
	redisCache, err := cache.NewCache(cache.CacheTypeRedis,
		cache.WithRedisConfig("localhost:6379", "", "demo:", 0),
		cache.WithHashExpiry(30*time.Minute),
	)
	if err != nil {
		fmt.Println("Redis连接失败，回退到内存缓存:", err)
		redisCache = memCache
	}
	defer redisCache.Close()

	// ========== 2. 基础键值操作 ==========
	fmt.Println("\n=== 基础键值操作 ===")

	// 设置值（5分钟过期）
	key := "user:1001:name"
	if err := redisCache.Set(key, "张三", 5*time.Minute); err != nil {
		fmt.Println("设置缓存失败:", err)
	}

	// 获取值
	if val, exists, err := redisCache.Get(key); err == nil && exists {
		fmt.Printf("获取缓存: key=%s, value=%v\n", key, val)
	} else {
		fmt.Println("缓存不存在或获取失败:", err)
	}

	// ========== 3. 哈希表操作 ==========
	fmt.Println("\n=== 哈希表操作 ===")

	hashKey := "user:1001:profile"
	profile := map[string]interface{}{
		"name":   "李四",
		"age":    28,
		"active": true,
	}

	// 设置哈希表
	if err := redisCache.SetHash(hashKey, profile, time.Hour); err != nil {
		fmt.Println("设置哈希表失败:", err)
	}

	// 获取单个字段
	if age, err := redisCache.GetHashField(hashKey, "age"); err == nil {
		fmt.Printf("获取哈希字段: age=%s\n", age)
	}

	// 检查字段是否存在
	if exists, err := redisCache.ExistHash(hashKey, "active"); err == nil {
		fmt.Printf("字段存在检查: active=%v\n", exists)
	}

	// ========== 4. 性能对比 ==========
	fmt.Println("\n=== 性能对比 ===")

	testKey := "benchmark:test"
	start := time.Now()

	// 内存缓存操作
	for i := 0; i < 1000; i++ {
		_ = memCache.Set(fmt.Sprintf("%s:%d", testKey, i), i, 0)
	}
	fmt.Printf("内存缓存 1000次写入: %v\n", time.Since(start))

	// Redis缓存操作
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = redisCache.Set(fmt.Sprintf("%s:%d", testKey, i), i, 0)
	}
	fmt.Printf("Redis缓存 1000次写入: %v\n", time.Since(start))

	// ========== 5. 清理操作 ==========
	fmt.Println("\n=== 清理操作 ===")

	if err := redisCache.Delete(key); err == nil {
		fmt.Println("成功删除键:", key)
	}

	if err := redisCache.DelHash(hashKey, "age"); err == nil {
		fmt.Println("成功删除哈希字段: age")
	}
}
