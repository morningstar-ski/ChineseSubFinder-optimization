package v1

import (
	"net/http"

	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/gin-gonic/gin"
)

func (cb *ControllerBase) CheckSubSupplierHandler(c *gin.Context) {
	var err error
	defer func() {
		cb.ErrorProcess(c, "CheckSubSupplierHandler", err)
	}()

	req := backend.CheckSubSupplier{}
	if c.Request.ContentLength > 0 {
		err = c.ShouldBindJSON(&req)
		if err != nil {
			return
		}
	}

	outStatus := subSupplier.ProbeSupplierStatuses(cb.cronHelper.FileDownloader, req.SupplierNames)
	c.JSON(http.StatusOK, outStatus)
}
