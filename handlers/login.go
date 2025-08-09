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
	user := &models.User{
		Username: c.PostForm("username"),
		Password: c.PostForm("password"),
	}
	result := models.DB.Where("username = ?", user.Username).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println("User not found")
		models.DB.Create(&user)
		token, _ := utils.GenerateToken(c, user.ID, user.Username, user.IsAdmin)
		c.SetCookie("token", token, 3600, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/blog")
		return
	} else if result.Error != nil {
		log.Println(result.Error)
	} else if user.Password != c.PostForm("password") {
		c.JSON(401, gin.H{
			"error": "Invalid password",
		})
	} else {
		token, _ := utils.GenerateToken(c, user.ID, user.Username, user.IsAdmin)
		c.SetCookie("token", token, 3600, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/blog")
	}
}

func ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}
