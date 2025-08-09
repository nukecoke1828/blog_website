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

// 自定义的切片类型，必须实现 Scanner 和 Valuer 接口
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
	Tags     StringSlice `gorm:"type:json"`
	Comments []Comment   `gorm:"foreignKey:BlogID"`
}

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	BlogID    uint
	Blog      Blog
	AuthorID  uint
	Author    User   `gorm:"foreignKey:AuthorID;references:ID"`
	Content   string `gorm:"type:text"`
	CreatedAt time.Time
	Likes     Like `gorm:"foreignKey:CommentID;references:ID"`
	Liked     bool `gorm:"default:false"`
	LikeNum   uint `gorm:"default:0"`
	ParentID  *uint
	Parent    *Comment `gorm:"foreignKey:ParentID;references:ID"`
	// 回复列表（子评论）：通过ParentID关联当前评论的ID
	Replies []Comment `gorm:"foreignKey:ParentID;references:ID"`
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

type VisitLog struct {
	ID        uint `gorm:"primaryKey"`
	BlogID    uint
	IP        string
	UserAgent string
	CreatedAt time.Time
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
