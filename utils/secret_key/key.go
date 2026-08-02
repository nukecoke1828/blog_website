package utils

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

// GenerateSecureKey 生成指定字节数的安全随机字符串，返回 Base64 编码格式
func GenerateSecureKey(bytes int) string {
	b := make([]byte, bytes)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal("生成随机密钥失败:", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
