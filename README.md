# Go 缓存库

[![Go 参考文档](https://pkg.go.dev/badge/github.com/zjguoxin/goscache.svg)](https://pkg.go.dev/github.com/zjguoxin/goscache)
[![许可证: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

支持内存和 Redis 的 Go 缓存库

## 目录

## 📑 目录导航

- [✨ 核心特性](#核心特性)
- [📊 特性对比表](#特性对比表)
- [⚙️ 安装与要求](#安装与要求)
- [🚀 快速入门](#快速入门)
  - [基础缓存操作](#基础缓存操作)
  - [哈希表操作](#哈希表操作)
- [🔧 高级配置](#高级配置)
  - [内存缓存配置](#内存缓存配置)
  - [Redis 缓存配置](#redis缓存配置)
- [📋 API 参考](#api参考)
- [💡 最佳实践](#最佳实践)
- [❓ 常见问题](#常见问题)
- [🧪 测试指南](#测试)
- [📜 许可证](#许可证)

---

## <span id="核心特性">✨ 核心特性</span>

✅ **多存储后端**：支持内存和 Redis 两种存储方式  
✅ **简洁 API**：提供 Get/Set/Delete 等基础操作  
✅ **哈希表支持**：支持 Redis 风格的哈希表操作  
✅ **过期时间**：可为每个缓存项设置生存时间(TTL)

## <span id="特性对比表">📊 特性对比表</span>

| 特性               | 内存缓存 | Redis 缓存 |
| ------------------ | -------- | ---------- |
| 持久化             | ❌       | ✅         |
| 分布式支持         | ❌       | ✅         |
| 性能               | ⚡ 极快  | 🚀 快      |
| 内存限制           | 有       | 可配置     |
| 哈希表过期时间支持 | ✅       | ✅         |
| 批量操作支持       | ✅       | ✅         |

## <span id="安装与要求">⚙️ 安装与要求</span>

- Go 1.16+
- Redis Server 5.0+ (如果使用 Redis 服务)

```bash
go get github.com/zjguoxin/goscache@v1.0.0
```

## <span id="快速入门">🚀 快速入门</span>

### <span id="基础缓存操作">基础缓存操作</span>

```go
package main

import (
	"time"
	"github.com/zjguoxin/goscache/cache"
)

package main

import (
	"fmt"
	"time"

	"github.com/zjguoxin/goscache/cache"
)

func main() {
	// 初始化内存缓存（默认5分钟过期，10分钟清理间隔）
	memCache, err := cache.NewCache(cache.CacheTypeMemory)
	if err != nil {
		panic(err)
	}
	defer memCache.Close()

	// 设置缓存（10分钟过期）
	err = memCache.Set("username", "张三", 10*time.Minute)
	if err != nil {
		panic(err)
	}

	// 获取缓存
	if val, exists, err := memCache.Get("username"); err == nil && exists {
		fmt.Println("获取到:", val)
	}

	// 删除缓存
	err = memCache.Delete("username")
	if err != nil {
		panic(err)
	}
}
```

### <span id="哈希表操作">哈希表操作</span>

```go
// 初始化Redis缓存
redisCache, err := cache.NewCache(cache.CacheTypeRedis,
	cache.WithRedisConfig("localhost:6379", "", "cache:", 0),
	cache.WithHashExpiry(time.Hour),
)
if err != nil {
	panic(err)
}
defer redisCache.Close()

// 设置哈希表（1小时过期）
userData := map[string]interface{}{
	"name":  "李四",
	"email": "lisi@example.com",
	"age":   28,
}
err = redisCache.SetHash("user:1001", userData, time.Hour)
if err != nil {
	panic(err)
}

// 获取哈希字段
email, err := redisCache.GetHashField("user:1001", "email")
if err != nil {
	panic(err)
}
fmt.Println("用户邮箱:", email)

// 获取整个哈希表
userInfo, err := redisCache.GetHash("user:1001")
if err != nil {
	panic(err)
}
fmt.Println("用户信息:", userInfo)

// 删除哈希字段
err = redisCache.DelHash("user:1001", "email")
if err != nil {
    panic(err)
}
```

## <span id="高级配置">🔧 高级配置</span>

### <span id="内存缓存配置">内存缓存配置</span>

```go
// 自定义默认过期时间和默认清理间隔
memCache, err := cache.NewCache(cache.CacheTypeMemory,
	cache.WithExpiration(15*time.Minute, 30*time.Minute),
)
```

### <span id="redis缓存配置">Redis 缓存配置</span>

```go
redisCache, err := cache.NewCache(cache.CacheTypeRedis,
	cache.WithRedisConfig("redis.example.com:6379", "password", "app_prefix:", 1),
	cache.WithPoolConfig(200, 20),  // 连接池配置
	cache.WithHashExpiry(2*time.Hour),  // 哈希表默认过期时间
)
```

## <span id="api参考">📋 API 参考</span>

| 方法签名                                                             | 描述                 | 参数                                                                      | 返回值                                      |
| -------------------------------------------------------------------- | -------------------- | ------------------------------------------------------------------------- | ------------------------------------------- |
| `Set(key string, value interface{}, expiration time.Duration) error` | 设置键值对           | `key`: 键名<br>`value`: 存储值<br>`expiration`: 过期时间(-1 表示永不过期) | `error`: 错误信息                           |
| `Get(key string) (interface{}, bool)`                                | 获取键值             | `key`: 键名                                                               | `interface{}`: 获取的值<br>`bool`: 是否存在 |
| `Delete(key string)`                                                 | 删除键值             | `key`: 键名                                                               | -                                           |
| `Exists(key string)`                                                 | 检查键值是否存在     | `key`: 键名                                                               | `bool`: 是否存在                            |
| `SetHash(key string, value map[string]interface{}) error`            | 设置哈希表           | `key`: 哈希表键名<br>`value`: 哈希表数据(map)                             | `error`: 错误信息                           |
| `GetHashField(key string, field string) (interface{}, error)`        | 获取哈希字段值       | `key`: 哈希表键名<br>`field`: 字段名                                      | `interface{}`: 字段值<br>`error`: 错误信息  |
| `DelHash(key, field string) error`                                   | 删除哈希字段         | `key`: 哈希表键名<br>`field`: 字段名                                      | `error`: 错误信息                           |
| `ExistHash(key, field string) bool`                                  | 检查哈希字段是否存在 | `key`: 哈希表键名<br>`field`: 字段名                                      | `bool`: 是否存在                            |

**注意**：所有方法都是线程安全的

## <span id="最佳实践">💡 最佳实践</span>

```go
func ExampleUserSession() {
    // 初始化Redis缓存
    c, err := cache.InitCache("redis", "localhost:6379", "", 1, "myproject_cache:")
    if err != nil {
        panic(err)
    }

    // 用户登录
    session := map[string]interface{}{
        "userID": 1001,
        "token": "abc123xyz",
        "expire": time.Now().Add(24*time.Hour).Unix(),
    }

    // 存储会话(30分钟过期)
    if err := c.SetHash("session:abc123", session); err != nil {
        panic(err)
    }

    // 获取会话
    if token, err := c.GetHashField("session:abc123", "token"); err == nil {
        fmt.Println("当前会话token:", token)
    }
}
```

## <span id="常见问题">❓ 常见问题</span>

### 如何选择内存缓存还是 Redis 缓存？

- 内存缓存：适合单机应用、临时数据缓存、高性能场景
- Redis 缓存：适合分布式系统、需要持久化的数据、多服务共享缓存

### 为什么 Get 返回 interface{}类型？

- 为了支持存储任意类型值，使用时需要进行类型断言：

```go
if val, exists, err := cache.Get("key"); exists && err == nil {
	if str, ok := val.(string); ok {
		// 使用字符串值
	}
}
```

## <span id="测试指南">🧪 测试指南</span>

```bash
// 运行测试
go test -v ./...
//性能基准测试
go test -bench=. -benchmem
```

## <span id="许可证">📜 许可证</span>

[MIT](https://github.com/zjguoxin/goscache/blob/main/LICENSE)© zjguoxin

### 作者

[zjguoxin@163.com](https://github.com/zjguoxin)
