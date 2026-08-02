package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nukecoke1828/my_blog_website/config"
	"github.com/nukecoke1828/my_blog_website/models"
)

const (
	RefreshTokenExpireDuration = 7 * 24 * time.Hour // Refresh Token: 7天
)

type Claims struct {
	UserID   uint
	Username string
	IsAdmin  bool
	jwt.RegisteredClaims
}

func GenerateToken(UserID uint, Username string, IsAdmin bool) (tokenString string, err error) {
	claims := Claims{
		UserID:   UserID,
		Username: Username,
		IsAdmin:  IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString([]byte(config.JWT_KEY.JWT.AccessTokenSecret)) // 真正的JWTtoken
	// if err != nil { // 在工具函数中返回 HTTP 响应违反了分层架构原则
	// 	ctx.JSON(400, gin.H{"error": "Failed to generate token"})
	// 	return "", err
	// }
	// ctx.Set("token", tokenString) // 工具函数不应该处理 HTTP 响应或设置 Gin 上下文
	return tokenString, nil
}

func AuthJWTToken(tokenValue string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(token *jwt.Token) (i interface{}, e error) {
		return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}

// GenerateRefreshTokenString 生成随机的 RefreshToken 字符串及其 SHA256 哈希
func GenerateRefreshTokenString() (token string, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	hash = HashToken(token)
	return
}

// HashToken 对 token 字符串进行 SHA256 哈希
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// RefreshTokenPair 验证 RefreshToken 并生成新的令牌对
// 新的 RefreshToken 保持与旧的相同的过期时间，实现全局过期时间管理
func RefreshTokenPair(refreshTokenStr string) (newAccessToken string, newRefreshTokenStr string, err error) {
	hash := HashToken(refreshTokenStr)

	var rt models.RefreshToken
	if result := models.DB.Where("token = ? AND revoked = ?", hash, false).First(&rt); result.Error != nil {
		return "", "", errors.New("refresh token not found or revoked")
	}

	if time.Now().After(rt.ExpiresAt) {
		return "", "", jwt.ErrTokenExpired
	}

	// 获取用户信息
	var user models.User
	if result := models.DB.First(&user, rt.UserID); result.Error != nil {
		return "", "", result.Error
	}

	// 生成新的 AccessToken
	newAccessToken, err = GenerateToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return "", "", err
	}

	// 生成新的 RefreshToken（保持与旧的相同的过期时间）
	newRefreshTokenStr, newRefreshTokenHash, err := GenerateRefreshTokenString()
	if err != nil {
		return "", "", err
	}

	oldExpiresAt := rt.ExpiresAt

	// 使用事务：撤销旧的 RefreshToken 并创建新的
	tx := models.DB.Begin()

	if err := tx.Model(&rt).Update("revoked", true).Error; err != nil {
		tx.Rollback()
		return "", "", err
	}

	newRT := models.RefreshToken{
		UserID:    rt.UserID,
		Token:     newRefreshTokenHash,
		ExpiresAt: oldExpiresAt, // 保持相同的过期时间
	}
	if err := tx.Create(&newRT).Error; err != nil {
		tx.Rollback()
		return "", "", err
	}

	tx.Commit()

	return newAccessToken, newRefreshTokenStr, nil
}

// RevokeRefreshToken 撤销指定的 RefreshToken
func RevokeRefreshToken(refreshTokenStr string) error {
	hash := HashToken(refreshTokenStr)
	return models.DB.Model(&models.RefreshToken{}).Where("token = ?", hash).Update("revoked", true).Error
}

// ParseExpiredToken 从过期的 JWT Token 中提取 Claims（不验证过期时间）
// 用于在 RefreshToken 过期时，从过期的 AccessToken 中提取用户信息
func ParseExpiredToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}
	return nil, errors.New("invalid claims")
}

// setTokenCookies 设置 AccessToken 和 RefreshToken 的 Cookie，并将 RefreshToken 存入数据库
func SetTokenCookies(c *gin.Context, user *models.User) {
	// 生成 AccessToken（保持原有逻辑）
	token, _ := GenerateToken(user.ID, user.Username, user.IsAdmin)
	c.SetSameSite(http.SameSiteLaxMode) // CSRF防护
	c.SetCookie("token", token, 3600, "/", "localhost", false, true)

	// 生成并存储 RefreshToken
	refreshTokenStr, refreshTokenHash, err := GenerateRefreshTokenString()
	if err != nil {
		log.Println("Failed to generate refresh token:", err)
		return
	}

	rt := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenHash,
		ExpiresAt: time.Now().Add(RefreshTokenExpireDuration),
	}

	if err := models.DB.Create(&rt).Error; err != nil {
		log.Println("Failed to store refresh token:", err)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", refreshTokenStr, int(RefreshTokenExpireDuration.Seconds()), "/", "localhost", false, true)
}
