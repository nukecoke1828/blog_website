package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nukecoke1828/my_blog_website/assets"
	. "github.com/nukecoke1828/my_blog_website/handlers"
	. "github.com/nukecoke1828/my_blog_website/middleware"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils"
	"github.com/nukecoke1828/my_blog_website/utils/kafka"
	"github.com/nukecoke1828/my_blog_website/utils/taskrunner"
)

var MyownAccount *models.User = func() *models.User {
	hash, err := utils.EncryptPassword(os.Getenv("ADMIN_PASSWORD"))
	if err != nil {
		panic("failed to encrypt admin password: " + err.Error())
	}
	return &models.User{
		ID:       1,
		Username: "admin",
		Password: hash,
		IsAdmin:  true,
	}
}()

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	models.InitDB()
	err := models.InitRedis("", 0)
	if err != nil {
		panic(err)
	}
	result := models.DB.FirstOrCreate(MyownAccount, models.User{Username: MyownAccount.Username})
	if result.Error != nil {
		panic(result.Error)
	}
	kafka.Init(models.DB)
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")

	// 静态资源：使用 embed 嵌入到二进制中，彻底消除路径问题
	staticSub, _ := fs.Sub(assets.StaticFS, "static")
	router.StaticFS("/static", http.FS(staticSub))
	router.GET("/", HomeHandler)
	router.GET("/login", ShowLoginPage) // 用于渲染登录页面
	router.POST("/login", LoginHandler)
	router.GET("/profile", ProfileHandler)
	router.GET("/logout", LogoutHandler)
	router.GET("/blog", AuthHeadler, BlogHandler)
	router.GET("/blog/:id", AuthHeadler, GetBlogHandler)
	router.POST("/blog/:id/like", AuthHeadler, LikeBlogHandler)
	router.POST("/blog/:id/comment", AuthHeadler, CommentBlogHandler)
	router.POST("/comment/:id/like", AuthHeadler, LikeCommentHandler)
	router.POST("/comment/:id/reply", AuthHeadler, CommentCommentHandler)
	router.POST("/comment/:id/delete", AuthHeadler, DeleteCommentHandler)
	router.GET("/blog/create", AuthHeadlerAdmin, ShowCreatePage)
	router.GET("/blog/create/not_permit", NotPermitUserHandler)
	router.POST("/blog/create", AuthHeadlerAdmin, CreateBlogHandler)
	taskrunner.Start()
	// ✅ 关键修复：使用 http.Server 实现优雅退出
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 优雅关闭 HTTP（5秒超时,给还没有完成的任务最后5秒来执行）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// ✅ 关键修复：直接调用 Shutdown，不等待信号
	kafka.Shutdown()

	log.Println("Server exiting")
}
