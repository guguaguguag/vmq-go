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
	
	// 🛠️ 终极绝杀：强行在连接字符串末尾追加高版本 MySQL 所需的 TLS 加密及传统密码兼容参数
	fixedDsn := dsn + "&tls=skip-verify&allowNativePasswords=true"
	
	DB, err = gorm.Open(mysql.Open(fixedDsn), &gorm.Config{})
	if err != nil {
		err = fmt.Errorf("连接数据库失败：%s", err.Error())
	}
	return err
}

// 迁移数据库
func Migrate() error {
	err := DB.AutoMigrate(&PayOrder{}, &PayQrcode{}, &Setting{}, &Paylog{})
	return err
}

// 初始化数据
func initializeData() error {
	var settingCount int64
	if err := DB.Model(&Setting{}).Count(&settingCount).Error; err != nil {
		return err
	}
	if settingCount > 0 {
		return nil
	}
	settingData := map[string]string{
		"adminUser":     "admin",                            
		"adminPwd":      "21232f297a57a5a743894a0e4a801fc3", 
		"notifyUrl":     "",                                 
		"returnUrl":     "",                                 
		"apiSecret":     "",                                 
		"lastHeart":     "0",                                
		"lastPay":       "0",                                
		"expire":        "5",                                
		"orderType":     "1",                                
		"orderMaxNum":   "10",                               
		"wechatPay":     "",                                 
		"aliPay":        "",                                 
		"emailSMTPhost": "",                                 
		"emailSMTPport": "",                                 
		"emailSMTPuser": "",                                 
		"emailSMTPpwd":  "",                                 
		"emailSMTPfrom": "",                                 
		"emailSMTPto":   "",                                 
		"emailSMTPssl":  "1",                                
		"payNotice":     "0",                                
		"errorNotice":   "1",                                
		"monitorNotice": "1",                                
	}
	keys, data := utils.DictionaryOrderSort(settingData)
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		value := data[key]
		setting := Setting{
			VKey:   key,
			VValue: value,
		}
		if err := DB.Create(&setting).Error; err != nil {
			return err
		}
	}
	return nil
}

// 初始化数据库
func SetupDatabase(dsn string) error {
	if err := InitDB(dsn); err != nil {
		return err
	}
	if err := Migrate(); err != nil {
		return err
	}
	if err := initializeData(); err != nil {
		return err
	}
	return nil
}
