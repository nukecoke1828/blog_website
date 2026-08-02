package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils"
	"gorm.io/gorm"
)

func LoginHandler(c *gin.Context) {
	hashedPassword, err := utils.EncryptPassword(c.PostForm("password"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	user := &models.User{
		Username: c.PostForm("username"),
		Password: hashedPassword,
	}
	result := models.DB.Where("username = ?", user.Username).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println("User not found, creating new user")
		models.DB.Create(&user)
		utils.SetTokenCookies(c, user)
		c.Redirect(http.StatusSeeOther, "/blog")
		return
	} else if result.Error != nil {
		log.Println(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
		return
	} else if !utils.CheckPasswordHash(c.PostForm("password"), user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid password",
		})
		return
	} else {
		utils.SetTokenCookies(c, user)
		c.Redirect(http.StatusSeeOther, "/blog")
	}
}

func ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

// LogoutHandler 登出：撤销 RefreshToken 并清除 Cookie
func LogoutHandler(c *gin.Context) {
	// 撤销 RefreshToken
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err == nil && refreshTokenStr != "" {
		if err := utils.RevokeRefreshToken(refreshTokenStr); err != nil {
			log.Println("Failed to revoke refresh token:", err)
		}
	}
	c.SetSameSite(http.SameSiteLaxMode)
	// 清除 Cookie
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "localhost", false, true)

	// 跳转到登录页面
	c.Redirect(http.StatusFound, "/login")
}
