package models

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "root:123456@tcp(localhost:3306)/blogdb?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	err = DB.AutoMigrate(&User{}, &Blog{}, &Comment{}, &Like{}, &VisitLog{})
	if err != nil {
		log.Fatal("failed to migrate database")
	}
}
