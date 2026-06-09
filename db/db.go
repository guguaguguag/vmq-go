package db

import (
	"fmt"
	"vmq-go/utils"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// 初始化数据库连接
func InitDB(dsn string) error {
	var err error = nil
	
	// 🛠️ 终极绝杀组合：关闭 TLS 强求 + 开启公钥自动检索 + 允许传统密码
	// 这是解决标准云通道下 MySQL 8.0 密码错位最稳健的行业标准配置
	perfectDsn := "uocvrojp6blagnzz:p5fjVO41kucsF9tvSDyx@tcp(bjwxx0axwz3tph0vnhce-mysql.services.clever-cloud.com:3306)/bjwxx0axwz3tph0vnhce?charset=utf8mb4&parseTime=True&loc=Local&tls=false&allowPublicKeyRetrieval=true&allowNativePasswords=true"
	
	DB, err = gorm.Open(mysql.Open(perfectDsn
