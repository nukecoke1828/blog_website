package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nukecoke1828/my_blog_website/config"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils"
)

// refreshAuthToken 尝试用 RefreshToken 刷新令牌对
// 返回新的 AccessToken 和 RefreshToken，以及是否成功
func refreshAuthToken(c *gin.Context) (string, bool) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil || refreshTokenStr == "" {
		return "", false
	}

	newAccessToken, newRefreshTokenStr, err := utils.RefreshTokenPair(refreshTokenStr)
	if err != nil {
		return "", false
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", newAccessToken, 3600, "/", "localhost", false, true)
	c.SetCookie("refresh_token", newRefreshTokenStr, int(utils.RefreshTokenExpireDuration.Seconds()), "/", "localhost", false, true)

	return newAccessToken, true
}

func AuthHeadlerAdmin(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil || tokenString == "" {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// Token 过期，尝试刷新
			newAccessToken, ok := refreshAuthToken(c)
			if !ok {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			// 刷新成功，重新解析新 Token(使用返回的新accesstoken，而不是从cookie中获取，因为cookie中读取到的可能是旧的token)
			token, err = jwt.ParseWithClaims(newAccessToken, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
			})
			if err != nil {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
		} else {
			c.JSON(401, gin.H{"error": "parsing token failed"})
			c.Abort()
			return
		}
	}
	claims, ok := token.Claims.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		c.Abort()
		return
	}
	AdminUser := &models.User{
		Username: claims.Username,
		ID:       claims.UserID,
		IsAdmin:  claims.IsAdmin,
	}
	if !AdminUser.IsAdmin {
		c.Redirect(http.StatusFound, "/blog/create/not_permit")
		c.Abort()
		return
	}
	c.Set("AdminUser", AdminUser)
	c.Next()
}

func AuthHeadler(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil || tokenString == "" {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			newAccessToken, ok := refreshAuthToken(c)
			// Token 过期，尝试刷新
			if !ok {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			// 刷新成功，重新解析新 Token
			token, err = jwt.ParseWithClaims(newAccessToken, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.JWT_KEY.JWT.AccessTokenSecret), nil
			})
			if err != nil {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
		} else {
			c.JSON(401, gin.H{"error": "parsing token failed"})
			c.Abort()
			return
		}
	}
	claims, ok := token.Claims.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		c.Abort()
		return
	}
	username := claims.Username
	var user models.User
	if result := models.DB.Where("username = ?", username).First(&user); result.Error != nil {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	c.Set("User", user)
	c.Next()
}
