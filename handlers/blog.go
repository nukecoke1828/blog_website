package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nukecoke1828/my_blog_website/models"
	"gorm.io/gorm"
)

func CreateBlogHandler(c *gin.Context) {
	AdminUser, ok := c.Get("AdminUser")
	if !ok {
		c.JSON(401, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	if AdminUser.(*models.User).IsAdmin {
		Title := c.PostForm("title")
		Content := c.PostForm("content")
		Tags := c.PostForm("tags")
		if Title == "" || Content == "" {
			c.JSON(400, gin.H{"error": "Title and Content cannot be empty"})
			return
		}
		var tags models.StringSlice
		if Tags != "" {
			rawTags := strings.Split(Tags, ",")
			for _, t := range rawTags {
				t = strings.TrimSpace(t)
				if t != "" && t != "nil" {
					tags = append(tags, t)
				}
			}
			if len(tags) == 0 {
				tags = nil
			}
		} else {
			tags = nil
		}
		// 调试输出 tags
		log.Printf("CreateBlogHandler tags: %#v", tags)
		blog := models.Blog{
			Title:    Title,
			Content:  Content,
			AuthorID: AdminUser.(*models.User).ID, // 只需设置ID
			Tags:     tags,
		}
		if err := models.DB.Create(&blog).Error; err != nil { // ✅ 检查错误
			c.JSON(500, gin.H{"error": "数据库错误: " + err.Error()})
			return
		}
		c.Redirect(http.StatusSeeOther, "/blog")
		return
	}
}

func ShowCreatePage(c *gin.Context) {
	c.HTML(http.StatusOK, "blog_new.html", gin.H{})
}

func BlogHandler(c *gin.Context) {
	var blogs []models.Blog
	models.DB.Find(&blogs) // 或者根据用户ID筛选
	c.HTML(http.StatusOK, "blog_list.html", gin.H{
		"Blogs": blogs,
	})
}

func NotPermitUserHandler(c *gin.Context) {
	c.HTML(http.StatusForbidden, "no_permission", nil)
}

func GetBlogHandler(c *gin.Context) {
	// 从URL中获取博客ID
	ID := c.Param("id")
	var Blog *models.Blog
	// 预加载作者信息和点赞信息,否则无法获取作者信息和点赞数量
	result := models.DB.Preload("Author").Preload("Likes").First(&Blog, ID)
	if result.Error != nil {
		c.String(http.StatusNotFound, result.Error.Error())
		return
	}
	// 1. 查询所有主评论（ParentID = nil）
	var mainComments []models.Comment
	if err := models.DB.
		Preload("Author").
		Where("blog_id = ? AND parent_id IS NULL", Blog.ID).
		Order("created_at desc").
		Find(&mainComments).Error; err != nil {
		c.String(http.StatusInternalServerError, "查询评论失败: %v", err)
		return
	}

	// 2. 递归加载每个主评论的所有层级回复
	for i := range mainComments {
		replies, err := loadReplies(mainComments[i].ID)
		if err != nil {
			c.String(http.StatusInternalServerError, "查询回复失败: %v", err)
			return
		}
		mainComments[i].Replies = replies
	}
	// 手动加载 Likes
	var like models.Like
	models.DB.First(&like, "blog_id = ?", Blog.ID)
	Blog.Likes = like
	var liked bool
	mid, exist := c.Get("User")
	if exist {
		user := mid.(models.User)
		for _, uid := range Blog.Likes.UserID {
			if uid == user.ID {
				liked = true
				break
			}
		}
	}
	c.HTML(http.StatusOK, "blog_detail.html", gin.H{
		"Blog":     Blog,
		"Comments": mainComments,
		"Liked":    liked,                  // 是否点赞
		"LikeNum":  len(Blog.Likes.UserID), // 点赞总数
		"User":     mid,
	})
}

func LikeBlogHandler(c *gin.Context) {
	ID := c.Param("id")

	var blog models.Blog
	result := models.DB.Preload("Likes").First(&blog, ID)
	if result.Error != nil {
		c.String(http.StatusNotFound, "博客不存在")
		return
	}

	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能点赞")
		return
	}
	user := mid.(models.User)
	userID := user.ID
	var like models.Like
	err := models.DB.Where("blog_id = ?", blog.ID).First(&like).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有点赞记录，创建一条
		userIDs := models.UintSlice{userID}
		userIDsJson, _ := json.Marshal(userIDs)
		// 用 map 方式插入，保证 user_id 是 JSON 字符串
		err = models.DB.Model(&models.Like{}).Create(map[string]interface{}{
			"blog_id": blog.ID,
			"user_id": string(userIDsJson),
		}).Error
		if err != nil {
			c.String(http.StatusInternalServerError, "创建点赞记录失败: %v", err.Error())
			return
		}
	} else {
		// 有点赞记录，更新它
		found := false
		for i, uid := range like.UserID {
			if uid == userID {
				like.UserID = append(like.UserID[:i], like.UserID[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			like.UserID = append(like.UserID, userID)
		}

		userIDsJson, _ := json.Marshal(like.UserID)

		err = models.DB.Model(&like).Where("blog_id = ?", blog.ID).Update("user_id", userIDsJson).Error
		if err != nil {
			c.String(http.StatusInternalServerError, "更新点赞失败: %v", err.Error())
			return
		}
	}
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", blog.ID))
}

func CommentBlogHandler(c *gin.Context) {
	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能评论")
		return
	}
	user := mid.(models.User)
	userID := user.ID

	ID := c.Param("id")
	id, err := strconv.ParseUint(ID, 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "无效的博客ID")
		return
	}

	content := c.PostForm("content")
	if content == "" {
		c.String(http.StatusBadRequest, "评论内容不能为空")
		return
	}

	comment := &models.Comment{
		BlogID:    uint(id),
		AuthorID:  userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	err = models.DB.Create(comment).Error
	if err != nil {
		c.String(http.StatusInternalServerError, "评论失败")
		return
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", id))
}

func LikeCommentHandler(c *gin.Context) {
	var err error
	// 预加载评论 + 评论作者
	var comment models.Comment
	var index uint64
	index, err = strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "无效的评论ID")
		return
	}
	comment.ID = uint(index)
	err = models.DB.Preload("Author").Where("id = ?", comment.ID).Order("created_at desc").Find(&comment).Error
	if err != nil {
		c.String(http.StatusNotFound, "评论不存在")
		return
	}
	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能点赞")
		return
	}
	user := mid.(models.User)
	userID := user.ID
	// 手动加载 Likes
	var like models.Like
	err = models.DB.First(&like, "comment_id = ?", comment.ID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有点赞记录，创建一条
		userIDs := models.UintSlice{userID}
		userIDsJson, _ := json.Marshal(userIDs)
		// 用 map 方式插入，保证 user_id 是 JSON 字符串
		err = models.DB.Model(&models.Like{}).Create(map[string]interface{}{
			"comment_id": comment.ID,
			"user_id":    string(userIDsJson),
		}).Error
		if err != nil {
			c.String(http.StatusInternalServerError, "创建点赞记录失败: %v", err.Error())
			return
		}
		models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("liked", true)
		models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("like_num", gorm.Expr("like_num + ?", 1))
	} else {
		// 有点赞记录，更新它
		found := false
		models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("liked", false)
		models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("like_num", gorm.Expr("like_num - ?", 1))
		for i, uid := range like.UserID {
			if uid == userID {
				like.UserID = append(like.UserID[:i], like.UserID[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			like.UserID = append(like.UserID, userID)
			models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("liked", true)
			models.DB.Model(&models.Comment{}).Where("id = ?", comment.ID).Update("like_num", gorm.Expr("like_num + ?", 1))
		}

		userIDsJson, _ := json.Marshal(like.UserID)

		err = models.DB.Model(&like).Where("comment_id = ?", comment.ID).Update("user_id", userIDsJson).Error
		if err != nil {
			c.String(http.StatusInternalServerError, "更新点赞失败: %v", err.Error())
			return
		}
	}
	id := comment.BlogID
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", id))
}

func CommentCommentHandler(c *gin.Context) {
	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能评论")
		return
	}
	user := mid.(models.User)
	userID := user.ID

	// 获取要回复的评论ID（可以是主评论或任何层级的回复）
	parentIDStr := c.Param("id")
	index, err := strconv.ParseUint(parentIDStr, 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "无效的评论ID")
		return
	}
	parentID := uint(index)
	// 验证父评论是否存在（无论它是主评论还是回复）
	var parentComment models.Comment
	if err := models.DB.Where("id = ?", parentID).First(&parentComment).Error; err != nil {
		c.String(http.StatusNotFound, "要回复的评论不存在")
		return
	}

	content := c.PostForm("content")
	if content == "" {
		c.String(http.StatusBadRequest, "评论内容不能为空")
		return
	}

	// 创建回复（ParentID指向要回复的评论ID）
	reply := &models.Comment{
		BlogID:    parentComment.BlogID, // 继承父评论的博客ID
		AuthorID:  userID,
		Content:   content,
		CreatedAt: time.Now(),
		ParentID:  (*uint)(&parentID), // 关键：ParentID可以是任何评论的ID
	}

	if err := models.DB.Create(reply).Error; err != nil {
		c.String(http.StatusInternalServerError, "评论失败: %v", err)
		return
	}

	// 重定向回博客详情页
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", parentComment.BlogID))
}

// 定义递归查询回复的函数
func loadReplies(commentID uint) ([]models.Comment, error) {
	var replies []models.Comment
	// 按时间顺序查询当前评论的直接回复
	if err := models.DB.
		Preload("Author").
		Preload("Parent.Author").
		Where("parent_id = ?", commentID).
		Order("created_at asc"). // 保持时间顺序
		Find(&replies).Error; err != nil {
		return nil, err
	}

	// 递归加载每个回复的子回复
	for i := range replies {
		subReplies, err := loadReplies(replies[i].ID)
		if err != nil {
			return nil, err
		}
		// 将子回复附加到当前回复的Replies字段
		replies[i].Replies = subReplies
	}

	return replies, nil
}

func DeleteCommentHandler(c *gin.Context) {
	id := c.Param("id")
	index, err := strconv.ParseUint(id, 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "无效的评论ID")
		return
	}
	commentID := uint(index)

	var comment models.Comment
	// 预加载所有嵌套的子评论
	if err := models.DB.Preload("Replies").First(&comment, commentID).Error; err != nil {
		c.String(http.StatusNotFound, "评论不存在")
		return
	}

	// 验证用户权限
	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能删除评论")
		return
	}
	user := mid.(models.User)
	if user.ID != comment.AuthorID && !user.IsAdmin {
		c.String(http.StatusForbidden, "没有权限删除该评论")
		return
	}

	// 删除所有子评论（递归删除）
	if err := deleteCommentAndReplies(commentID); err != nil {
		c.String(http.StatusInternalServerError, "删除评论失败: %v", err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", comment.BlogID))
}

// 递归删除评论及其所有子评论
func deleteCommentAndReplies(commentID uint) error {
	var comment models.Comment
	if err := models.DB.Preload("Replies").First(&comment, commentID).Error; err != nil {
		return err
	}

	// 递归删除所有子评论
	for _, reply := range comment.Replies {
		if err := deleteCommentAndReplies(reply.ID); err != nil {
			return err
		}
	}

	// 删除当前评论
	if err := models.DB.Delete(&comment).Error; err != nil {
		return err
	}

	return nil
}
