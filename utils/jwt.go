package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint
	Username string
	IsAdmin  bool
	jwt.RegisteredClaims
}

func GenerateToken(ctx *gin.Context, UserID uint, Username string, IsAdmin bool) (tokenString string, err error) {
	claims := Claims{
		UserID:   UserID,
		Username: Username,
		IsAdmin:  IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // Token expires after 24 hours
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString([]byte("secret"))
	// if err != nil { // 在工具函数中返回 HTTP 响应违反了分层架构原则
	// 	ctx.JSON(400, gin.H{"error": "Failed to generate token"})
	// 	return "", err
	// }
	// ctx.Set("token", tokenString) // 工具函数不应该处理 HTTP 响应或设置 Gin 上下文
	return tokenString, nil
}

func AuthJWTToken(tokenValue string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(token *jwt.Token) (i interface{}, e error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
