package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const ConfigKEY = "/app/config"

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var JWT_KEY AppConfig = AppConfig{
	JWT: JWTConfig{
		AccessTokenSecret:  os.Getenv("JWT_ACCESS_SECRET"),
		RefreshTokenSecret: os.Getenv("JWT_REFRESH_SECRET"),
	},
}

type AppConfig struct {
	JWT JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	AccessTokenSecret  string `mapstructure:"access_token_secret"`
	RefreshTokenSecret string `mapstructure:"refresh_token_secret"`
}

type ConfigManager struct {
	client   *clientv3.Client
	config   *AppConfig
	mu       sync.RWMutex
	key      string           // etcd 中存储配置的 key
	callback func(*AppConfig) // 配置变更时的回调函数
}

// NewConfigManager 创建配置管理器
func NewConfigManager(endpoints []string, key string) (*ConfigManager, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints, // etcd 集群地址，如 []string{"localhost:2379"}
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 etcd 失败: %w", err)
	}

	cm := &ConfigManager{
		client: client,
		key:    key,
	}

	// 首次加载配置
	if err := cm.loadConfig(); err != nil {
		return nil, err
	}

	return cm, nil
}

// ===== 3. 配置读取（线程安全） =====

// GetConfig 获取当前配置（只读，线程安全）
func (cm *ConfigManager) GetConfig() *AppConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	// 返回副本，防止外部修改
	configCopy := *cm.config
	return &configCopy
}

// GetConfigUnsafe 获取配置指针（仅内部使用）
func (cm *ConfigManager) getConfigUnsafe() *AppConfig {
	return cm.config
}

// ===== 4. 配置加载与解析 =====

func (cm *ConfigManager) loadConfig() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cm.client.Get(ctx, cm.key)
	if err != nil {
		return fmt.Errorf("从 etcd 读取配置失败: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("etcd 中未找到配置 key: %s", cm.key)
	}

	var config AppConfig
	if err := json.Unmarshal(resp.Kvs[0].Value, &config); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}

	cm.mu.Lock()
	cm.config = &config
	cm.mu.Unlock()

	fmt.Printf("[Config] 配置已加载: %+v\n", config)
	return nil
}

// StartWatch 启动配置热更新监听
func (cm *ConfigManager) StartWatch(ctx context.Context) {
	watchChan := cm.client.Watch(ctx, cm.key)

	go func() {
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case clientv3.EventTypePut: // 配置被修改
					fmt.Printf("[Config] 检测到配置变更\n")
					if err := cm.loadConfig(); err != nil { // 加载新配置
						fmt.Printf("[Config] 更新配置失败: %v\n", err)
					} else {
						fmt.Printf("[Config] 配置已自动热更新\n")
					}

				case clientv3.EventTypeDelete: // 配置被删除
					fmt.Printf("[Config] 警告：配置被删除！\n")
				}
			}
		}
		fmt.Println("[Config] Watch 监听已停止")
	}()
}

// SetCallback 设置配置变更回调
func (cm *ConfigManager) SetCallback(cb func(*AppConfig)) {
	cm.callback = cb
}

// Close 关闭连接
func (cm *ConfigManager) Close() error {
	return cm.client.Close()
}

// updateConfig 更新配置并触发回调
func (cm *ConfigManager) UpdateConfig(data []byte) error {
	var newConfig AppConfig
	if err := json.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("解析新配置失败: %w", err)
	}
	// 写入 etcd
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cm.client.Put(ctx, cm.key, string(data))
	if err != nil {
		return fmt.Errorf("写入 etcd 失败: %w", err)
	}

	// 更新内存
	cm.mu.Lock()
	oldConfig := cm.config
	cm.config = &newConfig
	cm.mu.Unlock()

	// 触发回调，让业务层处理配置变更
	if cm.callback != nil {
		cm.callback(&newConfig)
	}

	fmt.Printf("[Config] 配置已热更新\n")
	fmt.Printf("  旧配置: %+v\n", oldConfig)
	fmt.Printf("  新配置: %+v\n", newConfig)

	return nil
}
