/**
 * @Author: guxline zjguoxin@163.com
 * @Date: 2025/7/1 16:29:58
 * @LastEditors: guxline zjguoxin@163.com
 * @LastEditTime: 2025/7/1 16:29:58
 * Description: Redis缓存实现
 * Copyright: Copyright (©) 2025 中易综服. All rights reserved.
 */
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client    *redis.Client
	ctx       context.Context
	keyPrefix string
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(config *CacheConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.URL,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:    client,
		ctx:       ctx,
		keyPrefix: config.Prefix,
	}, nil
}

// getFullKey 获取完整键名
func (r *RedisCache) getFullKey(key string) string {
	return r.keyPrefix + key
}

// Get 获取缓存值
func (r *RedisCache) Get(key string) (interface{}, bool, error) {
	fullKey := r.getFullKey(key)
	val, err := r.client.Get(r.ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis get failed: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(val, &result); err != nil {
		return nil, false, fmt.Errorf("json unmarshal failed: %w", err)
	}
	return result, true, nil
}

// Set 设置缓存值
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	fullKey := r.getFullKey(key)
	val, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	if expiration == -1 {
		return r.client.Set(r.ctx, fullKey, val, 0).Err()
	}
	return r.client.Set(r.ctx, fullKey, val, expiration).Err()
}

// Delete 删除缓存值
func (r *RedisCache) Delete(key string) error {
	fullKey := r.getFullKey(key)
	return r.client.Del(r.ctx, fullKey).Err()
}

// SetHash 设置哈希表
func (r *RedisCache) SetHash(key string, value map[string]interface{}, expiration time.Duration) error {
	fullKey := r.getFullKey(key)
	if err := r.client.HMSet(r.ctx, fullKey, value).Err(); err != nil {
		return fmt.Errorf("redis hmset failed: %w", err)
	}

	if expiration > 0 {
		return r.client.Expire(r.ctx, fullKey, expiration).Err()
	}
	return nil
}

// GetHash 获取整个哈希表
func (r *RedisCache) GetHash(key string) (map[string]string, error) {
	fullKey := r.getFullKey(key)
	return r.client.HGetAll(r.ctx, fullKey).Result()
}

// GetHashField 获取哈希表字段
func (r *RedisCache) GetHashField(key, field string) (string, error) {
	fullKey := r.getFullKey(key)
	val, err := r.client.HGet(r.ctx, fullKey, field).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("field %s not found in hash %s", field, key)
		}
		return "", fmt.Errorf("redis hget failed: %w", err)
	}
	return val, nil
}

// DelHash 删除哈希表字段
func (r *RedisCache) DelHash(key, field string) error {
	fullKey := r.getFullKey(key)
	return r.client.HDel(r.ctx, fullKey, field).Err()
}

// ExistHash 检查哈希表字段是否存在
func (r *RedisCache) ExistHash(key, field string) (bool, error) {
	fullKey := r.getFullKey(key)
	exists, err := r.client.HExists(r.ctx, fullKey, field).Result()
	if err != nil {
		return false, fmt.Errorf("redis hexists failed: %w", err)
	}
	return exists, nil
}

// ExpireHash 设置哈希表过期时间
func (r *RedisCache) ExpireHash(key string, expiration time.Duration) error {
	fullKey := r.getFullKey(key)
	return r.client.Expire(r.ctx, fullKey, expiration).Err()
}

// MSet 批量设置缓存值
func (r *RedisCache) MSet(values map[string]interface{}, expiration time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range values {
		fullKey := r.getFullKey(key)
		val, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("json marshal failed for key %s: %w", key, err)
		}

		if expiration == -1 {
			pipe.Set(r.ctx, fullKey, val, 0)
		} else {
			pipe.Set(r.ctx, fullKey, val, expiration)
		}
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

// MGet 批量获取缓存值
func (r *RedisCache) MGet(keys []string) (map[string]interface{}, error) {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.getFullKey(key)
	}

	vals, err := r.client.MGet(r.ctx, fullKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget failed: %w", err)
	}

	result := make(map[string]interface{}, len(keys))
	for i, key := range keys {
		if vals[i] != nil {
			var value interface{}
			if err := json.Unmarshal([]byte(vals[i].(string)), &value); err != nil {
				return nil, fmt.Errorf("json unmarshal failed for key %s: %w", key, err)
			}
			result[key] = value
		}
	}

	return result, nil
}

// Close 关闭Redis连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}
