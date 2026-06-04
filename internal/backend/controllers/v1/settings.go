package v1

import (
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"net/http"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/backend/reload_policy"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/video_scan_and_refresh_helper"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (cb *ControllerBase) SettingsHandler(c *gin.Context) {
	var err error
	defer func() {
		// 统一的异常处理
		cb.ErrorProcess(c, "SettingsHandler", err)
	}()

	switch c.Request.Method {
	case "GET":
		{
			// 回复没有密码的 settings
			c.JSON(http.StatusOK, settings.Get().GetNoPasswordSettings())
		}
	case "PUT":
		{
			// 修改设置，这里不允许修改密码
			reqSetupInfo := settings.Settings{}
			err = c.ShouldBindJSON(&reqSetupInfo)
			if err != nil {
				return
			}
			// 需要去除 user 的 password 信息再保存，也就是继承之前的 password 即可
			oldSettings := settings.Get()
			needRestart := reload_policy.NeedRestartHTTPServer(oldSettings, &reqSetupInfo)
			nowPassword := settings.Get().UserInfo.Password
			reqSetupInfo.UserInfo.Password = nowPassword
			err = settings.SetFullNewSettings(&reqSetupInfo)
			if err != nil {
				return
			}
			pkg.ResetWantedVideoExt()
			// ----------------------------------------
			// 设置接口的 API TOKEN
			if settings.Get().ExperimentalFunction.ApiKeySettings.Enabled == true {
				common.SetApiToken(settings.Get().ExperimentalFunction.ApiKeySettings.Key)
			} else {
				common.SetApiToken("")
			}
			// ----------------------------------------
			err = syncDebugMode(cb.cronHelper.Logger)
			if err != nil {
				return
			}
			// ----------------------------------------
			err = local_http_proxy_server.SetProxyInfo(settings.Get().AdvancedSettings.ProxySettings.GetInfos())
			if err != nil {
				return
			}
			err = cb.cronHelper.ReloadSettings()
			if err != nil {
				return
			}
			if cb.videoScanAndRefreshHelper != nil {
				cb.videoScanAndRefreshHelper.Cancel()
			}
			cb.videoScanAndRefreshHelper = video_scan_and_refresh_helper.NewVideoScanAndRefreshHelper(
				sub_formatter.GetSubFormatter(cb.cronHelper.Logger, settings.Get().AdvancedSettings.SubNameFormatter),
				cb.cronHelper.FileDownloader, nil)
			c.JSON(http.StatusOK, backend.ReplyCommon{Message: "Settings Save Success"})
			if needRestart {
				// 仅当路由层依赖的设置变更时才重启 HTTP server
				cb.restartSignal <- 1
			}
		}
	default:
		c.JSON(http.StatusNoContent, backend.ReplyCommon{Message: "Settings Request.Method Error"})
	}
}

func syncDebugMode(logger *logrus.Logger) error {
	if settings.Get().AdvancedSettings.DebugMode == true {
		if err := log_helper.WriteDebugFile(); err != nil {
			return err
		}
		logger.SetLevel(logrus.DebugLevel)
		return nil
	}

	if err := log_helper.DeleteDebugFile(); err != nil {
		return err
	}
	logger.SetLevel(logrus.InfoLevel)
	return nil
}
