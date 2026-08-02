package models

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

// InitRedis 初始化连接
func InitRedis(password string, db int) error {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,     // e.g. "localhost:6379"
		Password: password, // 无密码留空
		DB:       db,       // 默认0号库
		PoolSize: 10,       // 连接池大小
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis连接失败: %w", err)
	}
	return nil
}

// GetClient 暴露客户端（高级操作时使用）
func GetClient() *redis.Client {
	return rdb
}

// Set 设置字符串缓存，带过期时间
func Set(key string, value any, expiration time.Duration) error {
	// 将任意类型序列化为 JSON
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存并反序列化到目标对象
func Get(key string, dest any) (bool, error) {
	data, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // 缓存不存在
	}
	if err != nil {
		return false, err
	}
	err = json.Unmarshal([]byte(data), dest)
	return true, err
}

// Delete 删除缓存
func Delete(keys ...string) error {
	return rdb.Del(ctx, keys...).Err()
}

// Exists 判断key是否存在
func Exists(key string) (bool, error) {
	n, err := rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// Expire 重新设置过期时间
func Expire(key string, ttl time.Duration) error {
	return rdb.Expire(ctx, key, ttl).Err()
}
