package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils"
)

func AuthHeadlerAdmin(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil || tokenString == "" {
		c.JSON(401, gin.H{"error": "Missing token"})
		c.Abort()
		return
	}
	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		c.JSON(401, gin.H{"error": "parsing token failed"})
	}
	AdminUser := &models.User{
		Username: token.Claims.(*utils.Claims).Username,
		ID:       token.Claims.(*utils.Claims).UserID,
		IsAdmin:  token.Claims.(*utils.Claims).IsAdmin,
	}
	if !AdminUser.IsAdmin {
		c.Redirect(http.StatusFound, "/blog/create/not_permit")
		return
	} else {
		AdminUser.IsAdmin = true
	}
	c.Set("AdminUser", AdminUser)
	c.Next()
}

func AuthHeadler(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil || tokenString == "" {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		c.JSON(401, gin.H{"error": "parsing token failed"})
	}
	username := token.Claims.(*utils.Claims).Username
	if result := models.DB.Where("username = ?", username).First(&models.User{}); result.Error != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	var user models.User
	models.DB.Where("username = ?", username).First(&user)
	c.Set("User", user)
	c.Next()
}
