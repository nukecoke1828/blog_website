package main

import (
	"github.com/gin-gonic/gin"
	. "github.com/nukecoke1828/my_blog_website/handlers"
	. "github.com/nukecoke1828/my_blog_website/middleware"
	"github.com/nukecoke1828/my_blog_website/models"
)

var MyownAccount *models.User = &models.User{
	ID:       1,
	Username: "nukecoke1828",
	Password: "114514",
	IsAdmin:  true,
}

func main() {
	models.InitDB()
	result := models.DB.FirstOrCreate(MyownAccount, models.User{Username: MyownAccount.Username})
	if result.Error != nil {
		panic(result.Error)
	}
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", HomeHandler)
	router.GET("/login", ShowLoginPage) // 用于渲染登录页面
	router.POST("/login", LoginHandler)
	router.GET("/profile", ProfileHandler)
	router.GET("/blog", AuthHeadler, BlogHandler)
	router.GET("/blog/:id", AuthHeadler, GetBlogHandler)
	router.POST("/blog/:id/like", AuthHeadler, LikeBlogHandler)
	router.POST("/blog/:id/comment", AuthHeadler, CommentBlogHandler)
	router.POST("/comment/:id/like", AuthHeadler, LikeCommentHandler)
	router.POST("/comment/:id/reply", AuthHeadler, CommentCommentHandler)
	router.POST("comment/:id/delete", AuthHeadler, DeleteCommentHandler)
	router.GET("/blog/create", AuthHeadlerAdmin, ShowCreatePage)
	router.GET("/blog/create/not_permit", NotPermitUserHandler)
	router.POST("/blog/create", AuthHeadlerAdmin, CreateBlogHandler)
	router.Run(":8080")
}
