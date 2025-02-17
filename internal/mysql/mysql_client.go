package mysql

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// User 定义用户模型
type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Username string `gorm:"unique;not null" json:"username"`
	Password string `gorm:"not null" json:"password"`
	Nickname string `json:"nickname"`
}

// MySQLClient 定义 MySQL 客户端结构体
type MySQLClient struct {
	DB *gorm.DB
}

// NewMySQLClient 创建 MySQL 实例
func NewMySQLClient(dataSource string) (*MySQLClient, error) {
	//log.Printf("Using MySQL data source: %s", dataSource)
	db, err := gorm.Open(mysql.Open(dataSource), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&User{})
	if err != nil {
		return nil, err
	}
	return &MySQLClient{
		DB: db,
	}, nil
}
