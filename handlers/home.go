package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HomeHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Name":        "NukeCoke",
		"Email":       "imdxhd@gmail.com",
		"GitHub":      "http://github.com/nukecoke1828",
		"Description": "这是一个由go语言实现的个人博客网站，目前只实现了一些基础功能，后续会扩展如博客分页、图片上传、退出登录、优化性能等功能。如果你有好的建议或者想法，欢迎在GitHub上提交issue或者pull request。",
	})
}
