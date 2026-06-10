package api

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"vmq-go/db"
	"vmq-go/middleware"
	"vmq-go/router/api/admin"
	"vmq-go/task"
	"vmq-go/utils"
	"vmq-go/utils/captcha"
	"vmq-go/utils/hash"
	"vmq-go/utils/qrcode"

	"github.com/gin-gonic/gin"
)

// 初始化路由
func InitRouter(route *gin.Engine) {
	routeGroup := route.Group("/api")
	
	// ============================================================
	// 🚀 核心修复：把所有前端依赖的查询接口全部移到中间件上方，彻底免疫 200 污染！
	// ============================================================
	// 创建订单
	routeGroup.POST("/order", creatOrderHandler)
	// 查询订单详情
	routeGroup.GET("/order/:orderId", getOrderGetHandler)
	// 查询订单支付状态（🔥 强行移到这里，脱离中间件魔爪）
	routeGroup.GET("/order/:orderId/state", getOrderStateGetHandler)
	// qrcode
	routeGroup.GET("/qrcode", qrcodeGetHandler)

	// --------------- 🚪 以下接口继续由中间件统一包装 ---------------
	routeGroup.Use(middleware.JSONMiddleware())
	admin.SetupAdminRoutes(routeGroup)
	// 解析二维码
	routeGroup.POST("/qrcode", qrcodePostHandler)
	// 验证码
	routeGroup.GET("/captcha", captchaHandler)
	// 心跳
	routeGroup.GET("/appHeart", HeartHandler)
	// 收到推送
	routeGroup.GET("/appPush", AppPushHandler)
	
	routeGroup.Use(middleware.AuthMiddleware())
	// 重新回调订单
	routeGroup.PUT("/order/:orderId", reCallbackOrderHandler)
}

func qrcodeGetHandler(c *gin.Context) {
	content := c.Query("content")
	format := c.DefaultQuery("format", "json")
	if content == "" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "content is empty",
		})
		return
	}
	base64Str, err := qrcode.QrcodeFromStr(content)
	if err != nil {
		c.JSON(
			200,
			gin.H{
				"code": -1,
				"msg":  err.Error(),
			},
		)
		return
	}
	switch format {
	case "image":
		c.Writer.Header().Set("Content-Type", "image/png")
		c.Request.Header.Set("Content-Type", "image/png")
		buf, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			c.JSON(
				200,
				gin.H{
					"code": -1,
					"msg":  err.Error(),
				},
			)
			return
		}
		c.Writer.Write(buf)
	default:
		c.JSON(200, gin.H{"qrcode": fmt.Sprintf("data:image/png;base64,%s", base64Str)})
	}
}

func qrcodePostHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}
	src, err := file.Open()
	if err != nil {
		c.Error(err)
		return
	}
	defer src.Close()
	buf := make([]byte, file.Size)
	_, err = src.Read(buf)
	if err != nil {
		c.Set("code", http.StatusInternalServerError)
		c.Error(err)
		return
	}
	base64Str := base64.StdEncoding.EncodeToString(buf)
	content, err := qrcode.DecodeQrcodeFromStr(base64Str)
	if err != nil {
		c.Set("code", http.StatusInternalServerError)
		c.Error(err)
		return
	}
	c.Set("data", gin.H{"content": content})
}

type CreateOrderParams struct {
	PayId     string  `json:"payId" binding:"required"`
	Type      int     `json:"type" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	Sign      string  `json:"sign" binding:"required"`
	Param     string  `json:"param"`
	NotifyUrl string  `json:"notifyUrl"`
	ReturnUrl string  `json:"returnUrl"`
}

func creatOrderHandler(c *gin.Context) {
	task.CheckOrderExpire()
	heart := task.CheckHeart()
	if !heart {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "heart error",
		})
		return
	}
	var params CreateOrderParams
	if c.ContentType() == "application/x-www-form-urlencoded" {
		payId := c.PostForm("payId")
		typeStr := c.PostForm("type")
		priceStr := c.PostForm("price")
		signstr := c.PostForm("sign")
		param := c.PostForm("param")
		notifyUrl := c.PostForm("notifyUrl")
		returnUrl := c.PostForm("returnUrl")
		if payId == "" || typeStr == "" || priceStr == "" || signstr == "" {
			c.JSON(200, gin.H{
				"code": -1,
				"msg":  "param error",
			})
			return
		}
		typeInt, err := strconv.Atoi(typeStr)
		if err != nil {
			c.JSON(200, gin.H{
				"code": -1,
				"msg":  "type error",
			})
			return
		}
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			c.JSON(200, gin.H{
				"code": -1,
				"msg":  "price error",
			})
			return
		}
		params = CreateOrderParams{
			PayId:     payId,
			Type:      typeInt,
			Price:     priceFloat,
			Sign:      signstr,
			Param:     param,
			NotifyUrl: notifyUrl,
			ReturnUrl: returnUrl,
		}
		appConfig, err := db.GetAppConfig()
		if err != nil {
			c.Error(err)
			return
		}
		sign := hash.GetMD5Hash(payId + param + typeStr + priceStr + appConfig.APISecret)
		if sign != params.Sign {
			c.JSON(200, gin.H{
				"code": -1,
				"msg":  "sign error",
			})
			return
		}
	} else {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "content-type error",
		})
		return
	}
	_, err := db.GetPayOrderByPayID(params.PayId)
	if err == nil || err.Error() != "record not found" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "payId is exist",
		})
		return
	}
	err = nil
	err = db.AddPayOrder(params.PayId, params.Type, params.Price, params.Param, params.NotifyUrl, params.ReturnUrl)
	if err != nil {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	order, err := db.GetPayOrderByPayID(params.PayId)
	if err != nil {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	timeout := (order.ExpectDate - order.CreateDate) / 60
	c.IndentedJSON(200, gin.H{
		"code": 1,
		"msg":  "success",
		"data": gin.H{
			"payId":       order.PayID,
			"orderId":     order.OrderID,
			"payType":     order.Type,
			"price":       order.Price,
			"reallyPrice": order.ReallyPrice,
			"payUrl":      order.PayURL,
			"isAuto":      order.IsAuto,
			"state":       order.State,
			"createDate":  order.CreateDate * 1000, 
			"expectDate":  order.ExpectDate * 1000,
			"timeOut":     timeout,
			"redirectUrl": fmt.Sprintf("/payment/%s", order.OrderID),
		},
	})
}

func getOrderGetHandler(c *gin.Context) {
	orderId := c.Param("orderId")
	if orderId == "" {
		c.IndentedJSON(200, gin.H{"code": -1, "msg": "orderId is empty"})
		return
	}
	order, err := db.GetPayOrderByOrderID(orderId)
	if err != nil {
		c.IndentedJSON(200, gin.H{"code": -1, "msg": err.Error()})
		return
	}
	timeout := (order.ExpectDate - order.CreateDate) / 60
	c.IndentedJSON(200, gin.H{
		"code": 1,
		"msg":  "success",
		"data": gin.H{
			"payId":       order.PayID,
			"orderId":     order.OrderID,
			"payType":     order.Type,
			"price":       order.Price,
			"reallyPrice": order.ReallyPrice,
			"payUrl":      order.PayURL,
			"isAuto":      order.IsAuto,
			"state":       order.State,
			"createDate":  order.CreateDate * 1000, 
			"expectDate":  order.ExpectDate * 1000, 
			"timeOut":     timeout,
		},
	})
}

// 🛠️ 终极修复：绕过中间件，给前端异步轮询脚本直出 code: 1 状态包！
func getOrderStateGetHandler(c *gin.Context) {
	orderId := c.Param("orderId")
	if orderId == "" {
		c.IndentedJSON(200, gin.H{"code": -1, "msg": "orderId is empty"})
		return
	}
	order, err := db.GetPayOrderByOrderID(orderId)
	if err != nil {
		if err.Error() == "record not found" {
			c.IndentedJSON(200, gin.H{"code": -1, "msg": "order not found"})
		} else {
			c.IndentedJSON(200, gin.H{"code": -1, "msg": err.Error()})
		}
		return
	}
	paramMap := map[string]string{
		"payId":       order.PayID,
		"param":       order.Param,
		"type":        fmt.Sprintf("%d", order.Type),
		"price":       utils.Float64ToSting(order.Price),
		"reallyPrice": utils.Float64ToSting(order.ReallyPrice),
	}
	appConfig, err := db.GetAppConfig()
	if err != nil {
		c.IndentedJSON(200, gin.H{"code": -1, "msg": err.Error()})
		return
	}
	sign := hash.GetMD5Hash(fmt.Sprintf("%s%s%s%s%s", order.PayID, order.Param, fmt.Sprintf("%d", order.Type), utils.Float64ToSting(order.Price), utils.Float64ToSting(order.ReallyPrice)) + appConfig.APISecret)
	paramStr := ""
	for k, v := range paramMap {
		paramStr += fmt.Sprintf("%s=%s&", k, v)
	}
	paramStr += fmt.Sprintf("sign=%s", sign)
	returnUrl := order.ReturnURL
	if returnUrl == "" {
		returnUrl = appConfig.ReturnUrl
	}
	var state int
	if order.State >= 1 {
		state = 1
		returnUrl = fmt.Sprintf("%s?%s", returnUrl, paramStr)
	} else {
		state = order.State
		returnUrl = ""
	}
	
	// 🚀 精准锁死 code: 1，喂饱前端轮询脚本，促使网页自动强行跳出！
	c.IndentedJSON(200, gin.H{
		"code": 1,
		"msg":  "success",
		"data": gin.H{
			"state":     state,
			"returnUrl": returnUrl,
		},
	})
}

func reCallbackOrderHandler(c *gin.Context) {
	orderId := c.Param("orderId")
	if orderId == "" {
		c.Error(fmt.Errorf("orderId is empty"))
		return
	}
	order, err := db.GetPayOrderByOrderID(orderId)
	if err != nil {
		c.Error(err)
		return
	}
	if order.State != 1 {
		c.Error(fmt.Errorf("order state error"))
		return
	}
	task.Notify(order)
	c.Set("code", http.StatusOK)
}

func captchaHandler(c *gin.Context) {
	id, b64s, err := captcha.GenerateCaptcha()
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", gin.H{
		"id":      id,
		"captcha": b64s,
	})
}

func HeartHandler(c *gin.Context) {
	time := c.Query("t")
	if time == "" {
		c.Error(fmt.Errorf("t is empty"))
		return
	}
	timeInt, err := strconv.ParseInt(time, 10, 64)
	if err != nil {
		c.Error(fmt.Errorf("time error"))
		return
	}
	timeNow := utils.GetUnix13()
	if math.Abs(float64(timeNow-timeInt)) > 10000 {
		c.Error(fmt.Errorf("time error"))
		return
	}
	sign := c.Query("sign")
	if sign == "" {
		c.Error(fmt.Errorf("sign is empty"))
		return
	}
	appConfig, err := db.GetAppConfig()
	if err != nil {
		c.Error(err)
		return
	}
	if hash.GetMD5Hash(time+appConfig.APISecret) != sign {
		c.Error(fmt.Errorf("sign error"))
		return
	}
	err = db.UpdateSetting("lastHeart", time)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("code", http.StatusOK)
	c.Set("data", "success")
}

func AppPushHandler(c *gin.Context) {
	t := c.Query("t")
	if t == "" {
		c.Error(fmt.Errorf("t is empty"))
		return
	}
	_type := c.Query("type") 
	if _type == "" {
		c.Error(fmt.Errorf("type is empty"))
		return
	}
	if _type != "1" && _type != "2" {
		c.Error(fmt.Errorf("type error"))
		return
	}
	typeInt, err := strconv.Atoi(_type)
	if err != nil {
		c.Error(err)
		return
	}
	price := c.Query("price")
	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		c.Error(err)
		return
	}
	sign := c.Query("sign")
	if sign == "" {
		c.Error(fmt.Errorf("sign is empty"))
		return
	}
	metdata := c.DefaultQuery("metadata", "")
	appConfig, err := db.GetAppConfig()
	if err != nil {
		c.Error(err)
		return
	}
	if hash.GetMD5Hash(_type+price+t+appConfig.APISecret) != sign {
		c.Error(fmt.Errorf("sign error"))
		return
	}
	go task.AppPush(typeInt, priceFloat, metdata)
	c.Set("code", http.StatusOK)
}
