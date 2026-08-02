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

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/nukecoke1828/my_blog_website/models"
	"github.com/nukecoke1828/my_blog_website/utils/kafka"
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
		invalidateBlogCache()
		c.Redirect(http.StatusSeeOther, "/blog")
		return
	}
}

func ShowCreatePage(c *gin.Context) {
	c.HTML(http.StatusOK, "blog_new.html", gin.H{"csrfField": csrf.TemplateField(c.Request)})
}

func BlogHandler(c *gin.Context) {
	// 从 URL 参数读取页码
	// 获取页码参数，默认第1页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize := 10 // 每页10条
	blogs, pagination, err := getPaginationBlog(page, pageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "获取博客列表失败: %v", err)
		return
	}
	c.HTML(http.StatusOK, "blog_list.html", gin.H{
		"Blogs":      blogs,
		"Pagination": pagination,
	},
	)
}

func NotPermitUserHandler(c *gin.Context) {
	c.HTML(http.StatusForbidden, "no_permission", gin.H{})
}

func GetBlogHandler(c *gin.Context) {
	// 从URL中获取博客ID
	ID := c.Param("id")
	var Blog models.Blog
	// 预加载作者信息和点赞信息,否则无法获取作者信息和点赞数量
	// Preload用来加载外键关联的表,例如这个语句会加载Blog的Author和Likes信息,因为Author和Likes都是Blog的外键关联
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

	// 2. 获取当前用户信息（用于点赞状态计算）
	var liked bool
	var currentUserID uint
	mid, exist := c.Get("User")
	if exist {
		user := mid.(models.User)
		currentUserID = user.ID
		// 判断博客是否已点赞
		for _, uid := range Blog.Likes.UserID {
			if uid == user.ID {
				liked = true
				break
			}
		}
	}

	// 3. 递归加载每个主评论的所有层级回复，并计算点赞状态
	for i := range mainComments {
		replies, err := loadReplies(mainComments[i].ID, currentUserID)
		if err != nil {
			c.String(http.StatusInternalServerError, "查询回复失败: %v", err)
			return
		}
		mainComments[i].Replies = replies
		// 加载该评论的点赞数据并计算状态
		loadCommentLikeData(&mainComments[i])
		setCommentLikeStatus(&mainComments[i], currentUserID)
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
	// 这个查询是多余的，因为我们在查询博客时已经预加载了点赞信息，直接使用 blog.Likes 即可
	err := models.DB.Where("blog_id = ?", blog.ID).First(&like).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有点赞记录，创建一条（user_id 是 JSON 列，需要显式序列化）
		userIDsJson, _ := json.Marshal(models.UintSlice{userID})
		err = models.DB.Model(&models.Like{}).Create(map[string]interface{}{
			"blog_id": blog.ID,
			"user_id": string(userIDsJson),
		}).Error
		if err != nil {
			c.String(http.StatusInternalServerError, "创建点赞记录失败: %v", err.Error())
			return
		}
	} else if err != nil {
		c.String(http.StatusInternalServerError, "查询点赞记录失败: %v", err.Error())
		return
	} else {
		// 有点赞记录，切换点赞状态
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
		err = models.DB.Model(&like).Where("blog_id = ?", blog.ID).Update("user_id", string(userIDsJson)).Error
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
		MsgID:     uuid.New().String(),
		BlogID:    uint(id),
		AuthorID:  userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	err = kafka.Producer().SendComment(c.Request.Context(), comment)
	if err != nil {
		c.String(http.StatusInternalServerError, "评论失败: %v", err)
		return
	}
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", id))
}

func LikeCommentHandler(c *gin.Context) {
	// 验证评论ID
	index, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "无效的评论ID")
		return
	}
	commentID := uint(index)

	// 验证评论存在
	var comment models.Comment
	if err := models.DB.Where("id = ?", commentID).First(&comment).Error; err != nil {
		c.String(http.StatusNotFound, "评论不存在")
		return
	}

	// 验证用户登录
	mid, exist := c.Get("User")
	if !exist {
		c.String(http.StatusUnauthorized, "未登录用户不能点赞")
		return
	}
	user := mid.(models.User)
	userID := user.ID

	// 查询或创建该评论的点赞记录（每条评论只有一条 Like 记录，UserID 切片存储所有点赞用户）
	var like models.Like
	// 应该使用 `Preload` 加载点赞信息，而不是在查询后手动加载
	err = models.DB.Where("comment_id = ?", comment.ID).First(&like).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 还没有点赞记录，创建一条（user_id 是 JSON 列，需要显式序列化）
		userIDsJson, _ := json.Marshal(models.UintSlice{userID})
		if err := models.DB.Model(&models.Like{}).Create(map[string]interface{}{
			"comment_id": comment.ID,
			"user_id":    string(userIDsJson),
		}).Error; err != nil {
			c.String(http.StatusInternalServerError, "创建点赞记录失败: %v", err.Error())
			return
		}
	} else if err != nil {
		c.String(http.StatusInternalServerError, "查询点赞记录失败: %v", err.Error())
		return
	} else {
		// 已有点赞记录，切换点赞状态
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
		if err := models.DB.Model(&like).Where("comment_id = ?", comment.ID).Update("user_id", string(userIDsJson)).Error; err != nil {
			c.String(http.StatusInternalServerError, "更新点赞失败: %v", err.Error())
			return
		}
	}
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/blog/%d", comment.BlogID))
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
		MsgID:     uuid.New().String(),  // 生成唯一幂等键
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
func loadReplies(commentID uint, currentUserID uint) ([]models.Comment, error) {
	var replies []models.Comment
	// 按时间顺序查询当前评论的直接回复，预加载当前回复的作者以及其父评论的作者
	if err := models.DB.
		Preload("Author").
		Preload("Parent.Author").
		Where("parent_id = ?", commentID). // 查询直接子评论
		Order("created_at asc").           // 保持时间顺序
		Find(&replies).Error; err != nil {
		return nil, err
	}

	// 递归加载每个回复的子回复
	for i := range replies {
		subReplies, err := loadReplies(replies[i].ID, currentUserID)
		if err != nil {
			return nil, err
		}
		replies[i].Replies = subReplies
		// 加载点赞数据并计算状态
		loadCommentLikeData(&replies[i])
		setCommentLikeStatus(&replies[i], currentUserID)
	}

	return replies, nil
}

// loadCommentLikeData 为评论单独加载点赞数据（避免 Preload 在 Like 表上的复杂性）
func loadCommentLikeData(comment *models.Comment) {
	var like models.Like
	if err := models.DB.Where("comment_id = ?", comment.ID).First(&like).Error; err == nil {
		comment.Likes = like
	}
}

// setCommentLikeStatus 根据 Like 记录和当前用户计算点赞状态
func setCommentLikeStatus(comment *models.Comment, currentUserID uint) {
	comment.LikeNum = len(comment.Likes.UserID)
	if currentUserID == 0 {
		comment.Liked = false
		return
	}
	for _, uid := range comment.Likes.UserID {
		if uid == currentUserID {
			comment.Liked = true
			return
		}
	}
	comment.Liked = false
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

func generatePages(current, total int) []int {
	var pages []int

	if total <= 7 {
		// 页数少，全部显示
		for i := 1; i <= total; i++ {
			pages = append(pages, i)
		}
		return pages
	}

	// 页数多，显示首尾+当前附近
	pages = append(pages, 1)

	if current > 3 {
		pages = append(pages, -1) // 省略号
	}

	start := max(2, current-1)
	end := min(total-1, current+1)

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	if current < total-2 {
		pages = append(pages, -1) // 省略号
	}

	pages = append(pages, total)
	return pages
}

func getPaginationBlog(page int, pageSize int) ([]models.Blog, *models.Pagination, error) {
	var blogs []models.Blog
	var total int64
	ok, err := models.Get(fmt.Sprint(page), &blogs)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		// 查询当前页数据
		offset := (page - 1) * pageSize
		if result := models.DB.Order("created_at desc").Limit(pageSize).Offset(offset).Find(&blogs); result.Error != nil {
			return nil, nil, result.Error
		}
		if len(blogs) > 0 {
			models.Set(fmt.Sprint(page), blogs, 10*time.Minute)
		} else {
			models.Set(fmt.Sprint(page), blogs, 1*time.Minute) // 空结果也缓存（防缓存穿透），但 TTL 设短一点
		}
	} else {
		fmt.Println("命中分页缓存")
	}
	ok, err = models.Get("total", &total)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		// 查询总数
		if result := models.DB.Model(&models.Blog{}).Count(&total); result.Error != nil {
			return nil, nil, result.Error
		}
		models.Set("total", &total, 10*time.Minute)
	} else {
		fmt.Println("命中总数缓存")
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	pagination := &models.Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		Total:       total,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		Pages:       generatePages(page, totalPages),
	}
	return blogs, pagination, nil
}

func invalidateBlogCache() {
	// 删除总数缓存
	if err := models.Delete("total"); err != nil {
		log.Printf("清除总数缓存失败: %v", err)
	}

	// 删除列表缓存：遍历前 20 页（足够覆盖大部分访问）
	// 如果你担心页数很多，可以把 pageSize 也加进循环，或者直接用 Redis Scan
	for i := 1; i <= 20; i++ {
		key := fmt.Sprint(i)
		if err := models.Delete(key); err != nil {
			log.Printf("清除列表缓存失败: %v", err)
		}
	}
}
