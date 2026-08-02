package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
)

func HomeHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"csrfField":   csrf.TemplateField(c.Request),
		"Name":        "NukeCoke",
		"Email":       "imdxhd@gmail.com",
		"GitHub":      "http://github.com/nukecoke1828",
		"Description": "这是一个由go语言+gin+gorm实现的个人博客网站，目前实现的功能有：登录、注册、退出登录（使用JWTtoken进行验证实现了accesstoken和refreshtoken,同时使用etcd+任务定时处理器实现jwt密钥的配置热更新）、查看文章列表、查看文章详情、写文章、文章点赞、发表评论（使用kafka异步落库）、评论点赞、嵌套评论以及分页查询（使用redis做分页缓存）,支持Docker一键部署，后续会扩展如图片视频上传、优化性能、文章编写支持markdown格式等功能。如果你有好的建议或者想法，欢迎在GitHub上提交issue或者pull request。",
	})
}
