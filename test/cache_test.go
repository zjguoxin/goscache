package cache_test

import (
	"fmt"
	"strconv"
	"sync"
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

	// 测试基础操作
	key := "test_key"
	value := "test_value"

	t.Run("SetAndGet", func(t *testing.T) {
		if err := c.Set(key, value, time.Minute); err != nil {
			t.Errorf("Set失败: %v", err)
		}

		if v, exists, err := c.Get(key); !exists || err != nil || v != value {
			t.Errorf("Get返回异常, 期望: %v, 实际: %v, 错误: %v", value, v, err)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := c.Exists(key)
		if !exists || err != nil {
			t.Errorf("Exists检测失败, 存在: %v, 错误: %v", exists, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := c.Delete(key); err != nil {
			t.Errorf("Delete失败: %v", err)
		}

		if _, exists, _ := c.Get(key); exists {
			t.Error("删除后键仍存在")
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		if err := c.Set(key, value, time.Second); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		if _, exists, _ := c.Get(key); exists {
			t.Error("键未按预期过期")
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := "concurrent_" + strconv.Itoa(i)
				_ = c.Set(key, i, time.Minute)
				_, _, _ = c.Get(key)
			}(i)
		}
		wg.Wait()
	})
}

func TestMemoryCache_Hash(t *testing.T) {
	c, err := cache.NewCache(cache.CacheTypeMemory)
	if err != nil {
		t.Fatalf("初始化内存缓存失败: %v", err)
	}
	defer c.Close()

	hashKey := "user:1001"
	userData := map[string]interface{}{
		"name":    "张三",
		"email":   "zhangsan@example.com",
		"age":     30,
		"active":  true,
		"balance": 100.50,
	}

	t.Run("SetAndGetHash", func(t *testing.T) {
		if err := c.SetHash(hashKey, userData, time.Minute); err != nil {
			t.Fatalf("SetHash失败: %v", err)
		}

		// 获取整个哈希表
		result, err := c.GetHash(hashKey)
		if err != nil {
			t.Fatalf("GetHash失败: %v", err)
		}

		if result["name"] != userData["name"] {
			t.Errorf("GetHash返回异常, 期望: %v, 实际: %v", userData["name"], result["name"])
		}

		// 获取单个字段
		email, err := c.GetHashField(hashKey, "email")
		if err != nil || email != userData["email"] {
			t.Errorf("GetHashField异常, 期望: %v, 实际: %v, 错误: %v", userData["email"], email, err)
		}

		// 检查字段存在性
		exists, err := c.ExistHash(hashKey, "name")
		if !exists || err != nil {
			t.Errorf("ExistHash检测失败, 存在: %v, 错误: %v", exists, err)
		}

		// 检查不存在的字段
		exists, err = c.ExistHash(hashKey, "nonexistent")
		if exists || err != nil {
			t.Errorf("ExistHash检测失败, 存在: %v, 错误: %v", exists, err)
		}
	})

	t.Run("HashExpiration", func(t *testing.T) {
		if err := c.SetHash(hashKey+"_exp", userData, time.Second); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		_, err := c.GetHash(hashKey + "_exp")
		if err == nil {
			t.Error("哈希表未按预期过期")
		}
	})

	t.Run("DelHash", func(t *testing.T) {
		if err := c.DelHash(hashKey, "email"); err != nil {
			t.Fatalf("DelHash失败: %v", err)
		}

		_, err := c.GetHashField(hashKey, "email")
		if err == nil {
			t.Error("删除后字段仍存在")
		}

		exists, err := c.ExistHash(hashKey, "email")
		if exists || err != nil {
			t.Errorf("删除后ExistHash检测失败, 存在: %v, 错误: %v", exists, err)
		}
	})

	t.Run("ExpireHash", func(t *testing.T) {
		if err := c.ExpireHash(hashKey, time.Second); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		_, err := c.GetHash(hashKey)
		if err == nil {
			t.Error("哈希表未按预期过期")
		}
	})
}

func TestRedisCache_Basic(t *testing.T) {
	c, err := cache.NewCache(cache.CacheTypeRedis,
		cache.WithRedisConfig("localhost:6379", "", "", 0),
	)
	if err != nil {
		t.Skip("Redis未运行，跳过测试")
	}
	defer c.Close()

	// 测试基础操作
	key := "test_key_redis"
	value := "test_value_redis"

	t.Run("SetAndGet", func(t *testing.T) {
		if err := c.Set(key, value, time.Minute); err != nil {
			t.Errorf("Set失败: %v", err)
		}

		if v, exists, err := c.Get(key); !exists || err != nil || v != value {
			t.Errorf("Get返回异常, 期望: %v, 实际: %v, 错误: %v", value, v, err)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := c.Exists(key)
		if !exists || err != nil {
			t.Errorf("Exists检测失败, 存在: %v, 错误: %v", exists, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := c.Delete(key); err != nil {
			t.Errorf("Delete失败: %v", err)
		}

		if _, exists, _ := c.Get(key); exists {
			t.Error("删除后键仍存在")
		}
	})

	t.Run("MSetAndMGet", func(t *testing.T) {
		items := map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		}

		if err := c.MSet(items, time.Minute); err != nil {
			t.Fatal(err)
		}

		keys := make([]string, 0, len(items))
		for k := range items {
			keys = append(keys, k)
		}

		result, err := c.MGet(keys)
		if err != nil {
			t.Fatal(err)
		}

		for k, expected := range items {
			actual := result[k]
			// 使用更智能的比较方式
			switch expected.(type) {
			case int:
				if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
					t.Errorf("MGet返回异常, 键: %s, 期望: %v (%T), 实际: %v (%T)",
						k, expected, expected, actual, actual)
				}
			default:
				if actual != expected {
					t.Errorf("MGet返回异常, 键: %s, 期望: %v (%T), 实际: %v (%T)",
						k, expected, expected, actual, actual)
				}
			}
		}
	})
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

	hashKey := "user:redis:1001"
	userData := map[string]interface{}{
		"name":    "李四",
		"email":   "lisi@example.com",
		"age":     25,
		"active":  true,
		"balance": 200.75,
	}

	t.Run("SetAndGetHash", func(t *testing.T) {
		if err := c.SetHash(hashKey, userData, time.Minute); err != nil {
			t.Fatalf("SetHash失败: %v", err)
		}

		result, err := c.GetHash(hashKey)
		if err != nil {
			t.Fatalf("GetHash失败: %v", err)
		}

		if result["name"] != userData["name"] {
			t.Errorf("GetHash返回异常, 期望: %v, 实际: %v", userData["name"], result["name"])
		}
	})

	t.Run("HashExpiration", func(t *testing.T) {
		tempKey := hashKey + "_exp"
		if err := c.SetHash(tempKey, userData, time.Second); err != nil {
			t.Fatal(err)
		}

		// 确认键存在
		exists, err := c.Exists(tempKey)
		if !exists || err != nil {
			t.Fatalf("设置后键不存在: %v", err)
		}

		time.Sleep(2 * time.Second)

		// 确认键已过期
		exists, err = c.Exists(tempKey)
		if exists || err != nil {
			t.Errorf("哈希表未按预期过期, 存在: %v, 错误: %v", exists, err)
		}
	})

	t.Run("ExpireHash", func(t *testing.T) {
		tempKey := hashKey + "_expire"
		if err := c.SetHash(tempKey, userData, time.Hour); err != nil {
			t.Fatal(err)
		}

		// 设置新的过期时间
		if err := c.ExpireHash(tempKey, time.Second); err != nil {
			t.Fatal(err)
		}

		time.Sleep(2 * time.Second)

		// 确认键已过期
		exists, err := c.Exists(tempKey)
		if exists || err != nil {
			t.Errorf("哈希表未按预期过期, 存在: %v, 错误: %v", exists, err)
		}
	})
}

func BenchmarkMemoryCache_Parallel(b *testing.B) {
	c, _ := cache.NewCache(cache.CacheTypeMemory)
	defer c.Close()

	b.Run("SetGet", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = c.Set("key", "value", time.Minute)
				_, _, _ = c.Get("key")
			}
		})
	})

	b.Run("HashOperations", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "hash_key_" + strconv.Itoa(i)
				_ = c.SetHash(key, map[string]interface{}{"field": i}, time.Minute)
				_, _ = c.GetHashField(key, "field")
				i++
			}
		})
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
			key := "hash_key_" + strconv.Itoa(i)
			_ = c.SetHash(key, map[string]interface{}{"field": i}, time.Minute)
			_, _ = c.GetHashField(key, "field")
		}
	})

	b.Run("MSetMGet", func(b *testing.B) {
		items := make(map[string]interface{}, 100)
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			key := "mset_key_" + strconv.Itoa(i)
			items[key] = "value_" + strconv.Itoa(i)
			keys[i] = key
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = c.MSet(items, time.Minute)
			_, _ = c.MGet(keys)
		}
	})
}
