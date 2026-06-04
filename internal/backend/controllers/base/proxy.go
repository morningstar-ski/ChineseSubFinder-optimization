package base

import (
	"net/http"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/gin-gonic/gin"
)

func (cb *ControllerBase) CheckProxyHandler(c *gin.Context) {
	var err error
	defer func() {
		cb.ErrorProcess(c, "CheckProxyHandler", err)
	}()

	if cb.proxyCheckLocker.Lock() == false {
		c.JSON(http.StatusOK, backend.ReplyCommon{Message: "running"})
		return
	}
	defer func() {
		cb.proxyCheckLocker.Unlock()
	}()

	checkProxy := backend.ReqCheckProxy{}
	err = c.ShouldBindJSON(&checkProxy)
	if err != nil {
		return
	}

	bkProxySettings := settings.Get().AdvancedSettings.ProxySettings.CopyOne()
	settings.Get().AdvancedSettings.ProxySettings = &checkProxy.ProxySettings
	settings.Get().AdvancedSettings.ProxySettings.UseProxy = true

	defer func() {
		settings.Get().AdvancedSettings.ProxySettings = bkProxySettings
		err = local_http_proxy_server.SetProxyInfo(settings.Get().AdvancedSettings.ProxySettings.GetInfos())
		if err != nil {
			return
		}
		local_http_proxy_server.GetProxyUrl()
	}()

	err = local_http_proxy_server.SetProxyInfo(settings.Get().AdvancedSettings.ProxySettings.GetInfos())
	if err != nil {
		return
	}

	outStatus := subSupplier.ProbeSupplierStatuses(cb.fileDownloader, nil)
	c.JSON(http.StatusOK, outStatus)
}
