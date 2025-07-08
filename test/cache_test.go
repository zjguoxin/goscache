package cache_test

import (
	"testing"
	"time"

	"github.com/zjguoxin/goscache/v2/cache"
)

func TestMemoryCache_Basic(t *testing.T) {
	c, err := cache.NewCache(cache.CacheTypeMemory)
	if err != nil {
		t.Fatalf("初始化内存缓存失败: %v", err)
	}
	defer c.Close()

	// 测试Set/Get
	key := "test_key"
	value := "test_value"
	if err := c.Set(key, value, time.Minute); err != nil {
		t.Errorf("Set失败: %v", err)
	}

	if v, exists, err := c.Get(key); !exists || err != nil || v != value {
		t.Errorf("Get返回异常, 期望: %v, 实际: %v, 错误: %v", value, v, err)
	}

	// 测试Delete
	if err := c.Delete(key); err != nil {
		t.Errorf("Delete失败: %v", err)
	}

	if _, exists, _ := c.Get(key); exists {
		t.Error("删除后键仍存在")
	}
}

func TestRedisCache_Hash(t *testing.T) {
	c, err := cache.NewCache(cache.CacheTypeRedis,
		cache.WithRedisConfig("localhost:6379", "", "", 0),
		cache.WithHashExpiry(time.Minute),
	)
	if err != nil {
		t.Skip("Redis未运行，跳过测试")
	}
	defer c.Close()

	// 测试哈希表
	hashKey := "user:1001"
	userData := map[string]interface{}{
		"name":  "张三",
		"email": "zhangsan@example.com",
	}

	// SetHash
	if err := c.SetHash(hashKey, userData, time.Minute); err != nil {
		t.Errorf("SetHash失败: %v", err)
	}

	// GetHashField
	email, err := c.GetHashField(hashKey, "email")
	if err != nil || email != userData["email"] {
		t.Errorf("GetHashField异常, 期望: %v, 实际: %v, 错误: %v", userData["email"], email, err)
	}

	// ExistHash
	exists, err := c.ExistHash(hashKey, "name")
	if !exists || err != nil {
		t.Errorf("ExistHash检测失败, 存在: %v, 错误: %v", exists, err)
	}

	// DelHash
	if err := c.DelHash(hashKey, "email"); err != nil {
		t.Errorf("DelHash失败: %v", err)
	}
}

func BenchmarkMemoryCache_Parallel(b *testing.B) {
	c, _ := cache.NewCache(cache.CacheTypeMemory)
	defer c.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = c.Set("key", "value", time.Minute)
			_, _, _ = c.Get("key")
		}
	})
}

func BenchmarkRedisCache_Operations(b *testing.B) {
	c, err := cache.NewCache(cache.CacheTypeRedis,
		cache.WithRedisConfig("localhost:6379", "", "", 0),
		cache.WithPoolConfig(200, 20),
	)
	if err != nil {
		b.Skip("Redis未运行，跳过基准测试")
	}
	defer c.Close()

	b.Run("SetGet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = c.Set("key", "value", time.Minute)
			_, _, _ = c.Get("key")
		}
	})

	b.Run("HashOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "hash_key"
			_ = c.SetHash(key, map[string]interface{}{"field": i}, time.Minute)
			_, _ = c.GetHashField(key, "field")
		}
	})
}
