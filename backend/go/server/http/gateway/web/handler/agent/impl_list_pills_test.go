// 旧服用列表读入口(GET /api/v1/agents/:uuid/pills)回归测试
// 任务 8 旧入口审计: 迁移后该路径读遗留 agent_pills 表(回滚保留),把已服用金丹
// 当作活跃绑定展示(移除能力后旧行仍在,与 effects 状态矛盾),且无任何调用方。
// 按「所有旧入口切换或关闭」恒 410 pill.legacy_api_removed;能力展示改走
// GET /api/v1/agents/:uuid/effects。
package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/service/agent_service"
	"github.com/alchemy-furnace/server/internal/service/pill_inventory_service"
	"github.com/alchemy-furnace/server/server/http/middleware"
	"github.com/alchemy-furnace/server/server/http/router"

	"github.com/gin-gonic/gin"
)

// setupListPillsRouter 注册旧服用列表路由(仅测本路由;memory 参数本测试未涉及,传 nil)
func setupListPillsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	h := New(agent_service.New(dao.NewAgentDao(), dao.NewModelDao(),
		pill_inventory_service.New(dao.GetDB(), time.Now)), nil)
	r.GET("/api/v1/agents/:uuid/pills", router.Wrapper(h.ListPills))
	return r
}

// getJSON 发送 GET 请求并解析响应包络
func getJSON(t *testing.T, r *gin.Engine, path string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应包络失败: %v, body: %s", err, w.Body.String())
	}
	return w.Code, envelope
}

// TestListPillsLegacyRemoved 合法道人 UUID 也 410: 遗留读入口已下线,不再返回旧绑定
func TestListPillsLegacyRemoved(t *testing.T) {
	db := setupTestDB(t)
	agentUUID, _ := seedAgentForBindPill(t, db)
	r := setupListPillsRouter()

	status, envelope := getJSON(t, r, fmt.Sprintf("/api/v1/agents/%s/pills", agentUUID))
	if status != http.StatusGone {
		t.Fatalf("期望 HTTP 410, 实际 %d, body: %v", status, envelope)
	}
	if ec, _ := envelope["error_code"].(string); ec != "pill.legacy_api_removed" {
		t.Fatalf("期望 error_code=pill.legacy_api_removed, 实际 %v", envelope["error_code"])
	}
}

// TestListPillsLegacyRemovedIgnoresPath 非法道人 UUID 同样 410(入口已死,不做解析)
func TestListPillsLegacyRemovedIgnoresPath(t *testing.T) {
	setupTestDB(t)
	r := setupListPillsRouter()

	status, envelope := getJSON(t, r, "/api/v1/agents/not-a-uuid/pills")
	if status != http.StatusGone {
		t.Fatalf("非法道人 UUID 期望 410, 实际 %d, body: %v", status, envelope)
	}
}
