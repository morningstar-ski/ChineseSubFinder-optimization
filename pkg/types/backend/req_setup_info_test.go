package backend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReqSetupInfoBindAllowsUnderscoreUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"settings":{"user_info":{"username":"audit_admin","password":"secret123"},"common_settings":{"movie_paths":["C:/tmp/m"],"series_paths":["C:/tmp/s"]}}}`
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	var info ReqSetupInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if info.Settings.UserInfo == nil {
		t.Fatal("user info is nil")
	}
	if info.Settings.UserInfo.Username != "audit_admin" {
		t.Fatalf("unexpected username: %q", info.Settings.UserInfo.Username)
	}
}
