/**
 * @Author: guxline zjguoxin@163.com
 * @Date: 2025/7/1 16:22:40
 * @LastEditors: guxline zjguoxin@163.com
 * @LastEditTime: 2025/7/1 16:22:40
 * Description: 内存缓存实现
 * Copyright: Copyright (©) 2025 中易综服. All rights reserved.
 */
package cache

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	cache             *cache.Cache
	hashMaps          map[string]map[string]interface{}
	hashExpirations   map[string]time.Time
	keyExpirations    map[string]time.Time
	mu                sync.RWMutex
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
	stopChan          chan struct{}
}

// NewMemoryCache 创建新的内存缓存实例
func NewMemoryCache(config *CacheConfig) (*MemoryCache, error) {
	m := &MemoryCache{
		cache:             cache.New(config.DefaultExp, config.CleanupInt),
		hashMaps:          make(map[string]map[string]interface{}),
		hashExpirations:   make(map[string]time.Time),
		keyExpirations:    make(map[string]time.Time),
		defaultExpiration: config.DefaultExp,
		cleanupInterval:   config.CleanupInt,
		stopChan:          make(chan struct{}),
	}

	// 启动后台清理协程
	go m.cleanupExpiredHashes()

	return m, nil
}

// cleanupExpiredHashes 定期清理过期的哈希表
func (m *MemoryCache) cleanupExpiredHashes() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for key, expiry := range m.hashExpirations {
				if now.After(expiry) {
					delete(m.hashMaps, key)
					delete(m.hashExpirations, key)
				}
			}
			m.mu.Unlock()
		case <-m.stopChan:
			return
		}
	}
}

// Get 获取缓存值
func (m *MemoryCache) Get(key string) (interface{}, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, found := m.cache.Get(key)
	return val, found, nil
}

// Set 设置缓存值
func (m *MemoryCache) Set(key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var exp time.Duration
	switch {
	case expiration == -1:
		exp = cache.NoExpiration
	case expiration == 0:
		exp = m.defaultExpiration
	default:
		exp = expiration
	}

	m.cache.Set(key, value, exp)
	if exp != cache.NoExpiration {
		m.keyExpirations[key] = time.Now().Add(exp)
	} else {
		delete(m.keyExpirations, key)
	}
	return nil
}

// Delete 删除缓存值
func (m *MemoryCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Delete(key)
	return nil
}

func (m *MemoryCache) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 检查过期时间（如果有）
	if expiry, exists := m.keyExpirations[key]; exists {
		if time.Now().After(expiry) {
			return false, nil
		}
	}

	// 2. 检查键是否存在
	_, found := m.cache.Get(key)
	return found, nil
}

// SetHash 设置哈希表
func (m *MemoryCache) SetHash(key string, value map[string]interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 初始化哈希表（原子性替换）
	newHash := make(map[string]interface{}, len(value))

	// 类型标记转换（与 Redis 方案一致）
	for field, val := range value {
		switch v := val.(type) {
		case bool:
			if v {
				newHash[field] = "bool:true"
			} else {
				newHash[field] = "bool:false"
			}
		case int, int32, int64, uint, uint32, uint64:
			newHash[field] = fmt.Sprintf("int:%v", v)
		case float32, float64:
			newHash[field] = fmt.Sprintf("float:%v", v)
		case string:
			newHash[field] = fmt.Sprintf("string:%s", v) // 明确标记字符串
		case []byte:
			newHash[field] = fmt.Sprintf("bytes:%x", v) // 二进制转十六进制
		default:
			// 复杂类型回退到 JSON
			jsonData, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("unsupported type for field %s: %w", field, err)
			}
			newHash[field] = fmt.Sprintf("json:%s", jsonData)
		}
	}

	// 原子性更新哈希表
	m.hashMaps[key] = newHash

	// 设置过期时间
	if expiration > 0 {
		m.hashExpirations[key] = time.Now().Add(expiration)
	} else if expiration == 0 && m.defaultExpiration > 0 {
		m.hashExpirations[key] = time.Now().Add(m.defaultExpiration)
	} else {
		delete(m.hashExpirations, key) // 永久有效
	}

	return nil
}

// GetHash 获取整个哈希表
func (m *MemoryCache) GetHash(key string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查过期
	if expiry, exists := m.hashExpirations[key]; exists && time.Now().After(expiry) {
		delete(m.hashMaps, key)
		delete(m.hashExpirations, key)
		return nil, fmt.Errorf("key expired")
	}

	// 获取原始数据
	rawHash, exists := m.hashMaps[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// 类型转换
	result := make(map[string]interface{}, len(rawHash))
	for field, markedVal := range rawHash {
		markedStr, ok := markedVal.(string)
		if !ok {
			result[field] = markedVal // 非字符串直接保留（如旧数据）
			continue
		}

		// 解析类型标记
		parts := strings.SplitN(markedStr, ":", 2)
		if len(parts) != 2 {
			result[field] = markedStr // 无标记则保持字符串
			continue
		}

		switch parts[0] {
		case "bool":
			result[field] = parts[1] == "true"
		case "int":
			val, _ := strconv.ParseInt(parts[1], 10, 64)
			result[field] = val
		case "float":
			val, _ := strconv.ParseFloat(parts[1], 64)
			result[field] = val
		case "string":
			result[field] = parts[1]
		case "bytes":
			data, _ := hex.DecodeString(parts[1])
			result[field] = data
		case "json":
			var data interface{}
			if err := json.Unmarshal([]byte(parts[1]), &data); err == nil {
				result[field] = data
			} else {
				result[field] = parts[1] // 解析失败保留原始 JSON
			}
		default:
			result[field] = markedStr // 未知标记保持原样
		}
	}

	return result, nil
}

// GetHashField 获取哈希表字段
func (m *MemoryCache) GetHashField(key, field string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查哈希表是否过期
	if expiry, exists := m.hashExpirations[key]; exists && time.Now().After(expiry) {
		return "", fmt.Errorf("hash key %s expired", key)
	}

	// 获取哈希表
	hash, exists := m.hashMaps[key]
	if !exists {
		return "", fmt.Errorf("hash key %s not found", key)
	}

	// 获取字段值
	markedVal, ok := hash[field]
	if !ok {
		return "", fmt.Errorf("field %s not found in hash %s", field, key)
	}

	// 解析带类型标记的值
	markedStr, ok := markedVal.(string)
	if !ok {
		return fmt.Sprintf("%v", markedVal), nil // 非字符串直接转为字符串
	}

	// 解析类型标记（格式为 "type:value"）
	parts := strings.SplitN(markedStr, ":", 2)
	if len(parts) != 2 {
		return markedStr, nil // 无类型标记则直接返回
	}

	// 根据类型返回原始值的字符串表示
	switch parts[0] {
	case "bool", "int", "float", "string", "bytes", "json":
		return parts[1], nil
	default:
		return markedStr, nil // 未知类型标记保持原样
	}
}

// DelHash 删除哈希表字段
func (m *MemoryCache) DelHash(key, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, exists := m.hashMaps[key]
	if !exists {
		return fmt.Errorf("hash key %s not found", key)
	}

	if _, ok := hash[field]; !ok {
		return fmt.Errorf("field %s not found in hash %s", field, key)
	}

	delete(hash, field)

	if len(hash) == 0 {
		delete(m.hashMaps, key)
		delete(m.hashExpirations, key)
	}

	return nil
}

// ExistHash 检查哈希表字段是否存在
func (m *MemoryCache) ExistHash(key, field string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if expiry, exists := m.hashExpirations[key]; exists && time.Now().After(expiry) {
		return false, fmt.Errorf("hash key %s expired", key)
	}

	hash, exists := m.hashMaps[key]
	if !exists {
		return false, nil
	}

	_, ok := hash[field]
	return ok, nil
}

// ExpireHash 设置哈希表过期时间
func (m *MemoryCache) ExpireHash(key string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.hashMaps[key]; !exists {
		return fmt.Errorf("hash key %s not found", key)
	}

	if expiration > 0 {
		m.hashExpirations[key] = time.Now().Add(expiration)
	} else {
		delete(m.hashExpirations, key)
	}

	return nil
}

// MSet 批量设置缓存值
func (m *MemoryCache) MSet(values map[string]interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var exp time.Duration
	switch {
	case expiration == -1:
		exp = cache.NoExpiration
	case expiration == 0:
		exp = m.defaultExpiration
	default:
		exp = expiration
	}

	for key, value := range values {
		m.cache.Set(key, value, exp)
	}

	return nil
}

// MGet 批量获取缓存值
func (m *MemoryCache) MGet(keys []string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		if val, found := m.cache.Get(key); found {
			result[key] = val
		}
	}

	return result, nil
}

// Close 关闭缓存，释放资源
func (m *MemoryCache) Close() error {
	close(m.stopChan)
	return nil
}
