package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProfileHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "profile.html", gin.H{
		"title":        "关于我",
		"name":         "吴承峰",
		"school":       "东莞理工学院 计算机专业",
		"hometown":     "广东省汕尾市海丰县",
		"hobbies":      []string{"街健", "打游戏", "编程"},
		"introduction": "你好，我是吴承峰，是东莞理工学院的计算机新生，我热爱学习计算机技术，正在搭建属于自己的博客网站。",
	})
}
