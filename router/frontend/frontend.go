package frontend

import (
	"os"
	"vmq-go/logger"
	"vmq-go/task"

	"github.com/gin-gonic/gin"
)

func SetupFrontendRoutes(router *gin.Engine) {
	// 检查web目录是否存在
	if _, err := os.Stat("./web"); os.IsNotExist(err) {
		logger.Logger.Info("web目录不存在，下载前端文件")
		// 下载前端文件
		task.DownloadFrontend()
	}
	// 挂载静态资源
	router.Static("/assets", "./web/assets")
	// 挂载 favicon.ico
	router.StaticFile("/favicon.ico", "./web/favicon.ico")
	// 返回 index.html
	router.StaticFile("/", "./web/index.html")

	// ============================================================
	// 🛠️ 核心修复：添加单页应用（SPA）历史模式路由万能兜底
	// ============================================================
	// 当粉丝访问 /payment/xxxx 等前端路由路径时，由于后端没有硬编码该门牌号，
	// 会无条件触发 NoRoute（路由未找到）机制。
	// 在这里我们强行把 index.html 吐回给粉丝的浏览器，让前端的 Vue/React 路由
	// 顺理成章地接管网址，并在原地秒级渲染出绚丽的“付款二维码收银台”！
	router.NoRoute(func(c *gin.Context) {
		c.File("./web/index.html")
	})
}
