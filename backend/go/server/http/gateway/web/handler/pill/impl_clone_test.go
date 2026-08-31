// 旧金丹克隆入口测试（任务 5 起恒 410 pill.legacy_api_removed）
// handler 不再解析路径/绑定请求体，也不再调用服务；任何载荷都返回 410。
package pill

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alchemy-furnace/server/server/http/router"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// setupCloneRouter 注册旧 clone 路由（任务 5 起恒 410）
func setupCloneRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(nil)
	r.POST("/api/v1/pills/:uuid/clone", router.Wrapper(h.Clone))
	return r
}

func performClone(t *testing.T, r *gin.Engine, path string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应包络失败: %v, body: %s", err, w.Body.String())
	}
	return w.Code, envelope
}

// TestCloneLegacyRemoved 合法路径也恒 410，且携带稳定错误码
func TestCloneLegacyRemoved(t *testing.T) {
	r := setupCloneRouter()

	status, envelope := performClone(t, r, "/api/v1/pills/"+uuid.NewString()+"/clone")

	if status != http.StatusGone {
		t.Fatalf("期望 HTTP 410, 实际 %d, body: %v", status, envelope)
	}
	if envelope["error_code"].(string) != "pill.legacy_api_removed" {
		t.Fatalf("error_code = %v, want pill.legacy_api_removed", envelope["error_code"])
	}
}

// TestCloneLegacyRemovedIgnoresPath 非法 UUID 路径同样 410（不再先解析再报 400）
func TestCloneLegacyRemovedIgnoresPath(t *testing.T) {
	r := setupCloneRouter()

	status, _ := performClone(t, r, "/api/v1/pills/not-a-uuid/clone")

	if status != http.StatusGone {
		t.Fatalf("期望 HTTP 410, 实际 %d", status)
	}
}
