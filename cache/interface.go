/**
 * @Author: guxline zjguoxin@163.com
 * @Date: 2025/7/1 16:21:29
 * @LastEditors: guxline zjguoxin@163.com
 * @LastEditTime: 2025/7/1 16:21:29
 * Description:
 * Copyright: Copyright (©) 2025 中易综服. All rights reserved.
 */
package cache

import (
	"fmt"
	"time"
)

type CacheType string

const (
	CacheTypeMemory CacheType = "memory"
	CacheTypeRedis  CacheType = "redis"

	defaultRedisURL        = "localhost:6379"
	defaultRedisPassword   = ""
	defaultRedisDB         = 0
	defaultRedisPrefix     = ""
	defaultExpiration      = 5 * time.Minute
	defaultCleanupInterval = 10 * time.Minute
	defaultPoolSize        = 100
	defaultMinIdleConns    = 10
)

type CacheConfig struct {
	Type          string        `json:"type"`            // 缓存类型: memory 或 redis
	URL           string        `json:"url"`             // Redis连接地址
	Password      string        `json:"password"`        // Redis密码
	DB            int           `json:"db"`              // Redis数据库索引
	Prefix        string        `json:"prefix"`          // Redis键前缀
	DefaultExp    time.Duration `json:"default_exp"`     // 默认过期时间
	CleanupInt    time.Duration `json:"cleanup_int"`     // 清理间隔(仅内存缓存)
	PoolSize      int           `json:"pool_size"`       // Redis连接池大小
	MinIdleConns  int           `json:"min_idle_conns"`  // Redis最小空闲连接数
	HashKeyExpiry time.Duration `json:"hash_key_expiry"` // 哈希表过期时间
}

type CacheInterface interface {
	// 基础操作
	Get(key string) (interface{}, bool, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Delete(key string) error
	Close() error

	// 哈希表操作
	SetHash(key string, value map[string]interface{}, expiration time.Duration) error
	GetHash(key string) (map[string]string, error)
	GetHashField(key, field string) (string, error)
	DelHash(key, field string) error
	ExistHash(key, field string) (bool, error)
	ExpireHash(key string, expiration time.Duration) error

	// 批量操作
	MSet(values map[string]interface{}, expiration time.Duration) error
	MGet(keys []string) (map[string]interface{}, error)
}

// Option 配置选项函数类型
type Option func(*CacheConfig)

// WithRedisConfig Redis配置选项
func WithRedisConfig(url, password, prefix string, db int) Option {
	return func(c *CacheConfig) {
		c.URL = url
		c.Password = password
		c.Prefix = prefix
		c.DB = db
	}
}

// WithExpiration 过期时间配置选项
func WithExpiration(defaultExp, cleanupInt time.Duration) Option {
	return func(c *CacheConfig) {
		c.DefaultExp = defaultExp
		c.CleanupInt = cleanupInt
	}
}

// WithPoolConfig 连接池配置选项
func WithPoolConfig(poolSize, minIdleConns int) Option {
	return func(c *CacheConfig) {
		c.PoolSize = poolSize
		c.MinIdleConns = minIdleConns
	}
}

// WithHashExpiry 哈希表过期时间配置选项
func WithHashExpiry(expiry time.Duration) Option {
	return func(c *CacheConfig) {
		c.HashKeyExpiry = expiry
	}
}

// InitCache 初始化缓存
// 参数:
// - 第一个参数: 缓存类型 (memory/redis)，可以是CacheType或字符串
// - 后续参数可以是以下任意组合:
//   - Redis连接URL (string)
//   - Redis密码 (string)
//   - 默认过期时间 (time.Duration)
//   - 清理间隔 (time.Duration)
//
// NewCache 创建缓存实例
func NewCache(cacheType CacheType, opts ...Option) (CacheInterface, error) {
	config := &CacheConfig{
		Type:          string(cacheType),
		URL:           defaultRedisURL,
		Password:      defaultRedisPassword,
		DB:            defaultRedisDB,
		Prefix:        defaultRedisPrefix,
		DefaultExp:    defaultExpiration,
		CleanupInt:    defaultCleanupInterval,
		PoolSize:      defaultPoolSize,
		MinIdleConns:  defaultMinIdleConns,
		HashKeyExpiry: 0, // 默认不设置过期时间
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	switch cacheType {
	case CacheTypeRedis:
		return NewRedisCache(config)
	case CacheTypeMemory:
		return NewMemoryCache(config)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
}
