package db

import (
	"fmt"
	"strings"
	"vmq-go/utils"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// 初始化数据库连接
func InitDB(dsn string) error {
	var err error = nil
	
	// 🛠️ 降维打击：直接硬编码 Clever Cloud 的纯正完整 DSN，彻底无视 config.yaml 的配置误差和空格干扰
	perfectDsn := "uocvrojp6blagnzz:p5fjVO41kucsF9tvSDyx@tcp(bjwxx0axwz3tph0vnhce-mysql.services.clever-cloud.com:3306)/bjwxx0axwz3tph0vnhce?charset=utf8mb4&parseTime=True&loc=Local&tls=skip-verify&allowNativePasswords=true&allowCleartextPasswords=true&allowPublicKeyRetrieval=true"
	
	DB, err = gorm.Open(mysql.Open(perfectDsn), &gorm.Config{})
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
