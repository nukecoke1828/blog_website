package taskrunner

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nukecoke1828/my_blog_website/config"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils"
	key "github.com/nukecoke1828/my_blog_website/utils/secret_key"
)

var ConfigClient *config.ConfigManager

var cancelWatch context.CancelFunc

func JWTkeyUpdateDispatcher(dc DataChan) error {
	accesskey := key.GenerateSecureKey(32)  // 生成32字节的随机密钥
	refreshkey := key.GenerateSecureKey(32) // 生成32字节的随机密钥
	JWTConfig := &config.JWTConfig{
		AccessTokenSecret:  accesskey,
		RefreshTokenSecret: refreshkey,
	}
	key, err := json.Marshal(JWTConfig)
	if err != nil {
		return err
	}
	dc <- key // 将生成的密钥发送到数据通道
	return nil
}

// 实现JWT密钥热更新的执行器函数
func JWTkeyUpdateExecutor(dc DataChan) error {
	makeEtcdConfigServer()
	defer ConfigClient.Close() // 关闭配置管理器
	defer cancelWatch()        // 关闭配置管理器并取消 Watch 监听
	data := <-dc
	err := ConfigClient.UpdateConfig(data.([]byte))
	if err != nil {
		log.Printf("Failed to update config via etcd: %v", err)
		return err
	}
	// StartWatch now reloads config directly from etcd, no channel signaling needed
	time.Sleep(5 * time.Second)
	return nil
}

func makeEtcdConfigServer() {
	// 从环境变量读取 etcd 地址，默认 localhost:12379（便于本地调试）
	endpoints := os.Getenv("ETCD_ENDPOINTS")
	if endpoints == "" {
		endpoints = "localhost:12379"
	}
	// 支持逗号分隔的多个地址，这里简单处理
	endpointList := strings.Split(endpoints, ",")
	var err error
	ConfigClient, err = config.NewConfigManager(endpointList, config.ConfigKEY)
	if err != nil {
		panic(err)
	}
	ConfigClient.SetCallback(func(newConfig *config.AppConfig) {
		// Revoke all active refresh tokens first (key rotation should invalidate old tokens)
		if err := models.DB.Model(&models.RefreshToken{}).
			Where("revoked = ?", false).
			Update("revoked", true).Error; err != nil {
			log.Println("Failed to revoke old refresh tokens:", err)
			return
		}

		// Query all users who have ever had a refresh token (distinct UserID)
		var userIDs []uint
		if err := models.DB.Model(&models.RefreshToken{}).
			Distinct("user_id").
			Pluck("user_id", &userIDs).Error; err != nil {
			log.Println("Failed to get distinct user IDs:", err)
			return
		}

		// Generate a unique refresh token for each user
		for _, uid := range userIDs {
			_, refreshTokenHash, err := utils.GenerateRefreshTokenString()
			if err != nil {
				log.Println("Failed to generate refresh token:", err)
				return
			}
			rt := models.RefreshToken{
				UserID:    uid,
				Token:     refreshTokenHash,
				ExpiresAt: time.Now().Add(utils.RefreshTokenExpireDuration),
			}
			if err := models.DB.Create(&rt).Error; err != nil {
				log.Println("Failed to store refresh token:", err)
				return
			}
		}
		log.Printf("Key rotation complete: revoked old tokens and issued new tokens for %d users", len(userIDs))
	})
	// 3. 启动 Watch 监听（核心：热更新）
	ctx, cancel := context.WithCancel(context.Background())
	cancelWatch = cancel
	ConfigClient.StartWatch(ctx)
}
