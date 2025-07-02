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
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	cache             *cache.Cache
	hashMaps          map[string]map[string]interface{}
	hashExpirations   map[string]time.Time
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
	return nil
}

// Delete 删除缓存值
func (m *MemoryCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Delete(key)
	return nil
}

// SetHash 设置哈希表
func (m *MemoryCache) SetHash(key string, value map[string]interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.hashMaps[key]; !exists {
		m.hashMaps[key] = make(map[string]interface{})
	}

	for field, val := range value {
		m.hashMaps[key][field] = val
	}

	if expiration > 0 {
		m.hashExpirations[key] = time.Now().Add(expiration)
	} else if expiration == 0 {
		m.hashExpirations[key] = time.Now().Add(m.defaultExpiration)
	}

	return nil
}

// GetHash 获取整个哈希表
func (m *MemoryCache) GetHash(key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if expiry, exists := m.hashExpirations[key]; exists && time.Now().After(expiry) {
		return nil, fmt.Errorf("hash key %s expired", key)
	}

	hash, exists := m.hashMaps[key]
	if !exists {
		return nil, fmt.Errorf("hash key %s not found", key)
	}

	result := make(map[string]string, len(hash))
	for k, v := range hash {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result, nil
}

// GetHashField 获取哈希表字段
func (m *MemoryCache) GetHashField(key, field string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if expiry, exists := m.hashExpirations[key]; exists && time.Now().After(expiry) {
		return "", fmt.Errorf("hash key %s expired", key)
	}

	hash, exists := m.hashMaps[key]
	if !exists {
		return "", fmt.Errorf("hash key %s not found", key)
	}

	val, ok := hash[field]
	if !ok {
		return "", fmt.Errorf("field %s not found in hash %s", field, key)
	}

	return fmt.Sprintf("%v", val), nil
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
