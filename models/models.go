package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

var _ CustomizationType = (*StringSlice)(nil)

var _ CustomizationType = (*UintSlice)(nil)

type StringSlice []string

type UintSlice []uint

// 自定义的切片类型，必须实现 Scanner 和 Valuer 接口(因为gorm不支持切片类型，需要使用json类型来存储)
type CustomizationType interface {
	Scan(src interface{}) error
	Value() (driver.Value, error)
}

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	IsAdmin  bool   `gorm:"default:false"`
}

type Blog struct {
	ID        uint   `gorm:"primaryKey"`
	Title     string `gorm:"not null"`
	Content   string `gorm:"type:text"`
	AuthorID  uint
	Author    User
	Likes     Like
	Visits    uint `gorm:"default:0"`
	CreatedAt time.Time
	// gorm不支持切片类型，需要使用json类型来存储
	Tags     StringSlice `gorm:"serializer:json"`
	Comments []Comment   `gorm:"foreignKey:BlogID"`
}

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	BlogID    uint      `json:"blog_id"`
	Blog      Blog      `json:"-"`
	AuthorID  uint      `json:"author_id"`
	Author    User      `gorm:"foreignKey:AuthorID;references:ID" json:"-"`
	MsgID     string    `gorm:"size:64;uniqueIndex:idx_comment_msg_id" json:"msg_id"` // 幂等键
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Likes     Like      `gorm:"foreignKey:CommentID;references:ID" json:"-"`
	ParentID  *uint     `json:"-"`
	Parent    *Comment  `gorm:"foreignKey:ParentID;references:ID" json:"-"`
	// 回复列表（子评论）：通过ParentID关联当前评论的ID
	Replies []Comment `gorm:"foreignKey:ParentID;references:ID" json:"-"`
	// 以下字段仅供模板渲染使用，不持久化到数据库
	Liked   bool `gorm:"-" json:"-"`
	LikeNum int  `gorm:"-" json:"-"`
}

type Like struct {
	ID        uint `gorm:"primaryKey"`
	BlogID    uint
	CommentID uint
	UserID    UintSlice `gorm:"type:json"`
}

func (l *Like) BeforeCreate(tx *gorm.DB) (err error) {
	if l.UserID == nil {
		l.UserID = make(UintSlice, 0)
	}
	return
}

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	Token     string    `gorm:"uniqueIndex;not null;size:512"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
}

type VisitLog struct {
	ID        uint `gorm:"primaryKey"`
	BlogID    uint
	IP        string
	UserAgent string
	CreatedAt time.Time
}

type Pagination struct {
	CurrentPage int   // 当前页码
	TotalPages  int   // 总页数
	Total       int64 // 总博客数
	HasPrev     bool  // 是否有上一页
	HasNext     bool  // 是否有下一页
	PrevPage    int   // 上一页页码
	NextPage    int   // 下一页页码
	Pages       []int // 页码列表（-1 表示省略号）
}

// 实现 Scanner 接口 - 从数据库读取时使用
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("不支持的数据库类型")
	}
	return json.Unmarshal(bytes, s)
}

// 实现 Valuer 接口 - 保存到数据库时使用(必须插入的是结构体才会调用)
func (s *StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *UintSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("不支持的数据库类型")
	}
	return json.Unmarshal(bytes, s)
}

func (s *UintSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}
