package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	// 从环境变量读取配置（若未设置则使用默认值）
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		panic("Failed to get database password from environment variable")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "blogdb"
	}

	// 1. 先连接到 MySQL 服务器（不指定具体数据库）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8&multiStatements=true",
		dbUser, dbPassword, dbHost, dbPort)
	tempDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err.Error())
	}
	defer tempDB.Close()

	// 2. 创建数据库
	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	_, err = tempDB.Exec(createSQL)
	if err != nil {
		panic("创建数据库失败: " + err.Error())
	}

	// 3. 连接到目标数据库
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	DB, err = gorm.Open(mysql.Open(targetDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}
	// 自动迁移数据库
	err = DB.AutoMigrate(&User{}, &Blog{}, &Comment{}, &Like{}, &VisitLog{}, &RefreshToken{})
	if err != nil {
		log.Fatal("failed to migrate database")
	}
}
