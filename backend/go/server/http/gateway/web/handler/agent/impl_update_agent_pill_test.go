// 更新服用记录接口回归测试(bug 修复: body 不再要求 pill_id,金丹由路径 UUID 标识)
// 使用 sqlite 内存库(glebarez/sqlite,纯 Go 驱动),无需外部基础设施
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/service/agent_service"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/middleware"
	"github.com/alchemy-furnace/server/server/http/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupTestDB 初始化 sqlite 内存库并注入新架构全局 dao.DB
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:agenttest%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开 sqlite 内存库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.DaoAgent{}, &model.ElixirPill{}, &model.AgentPill{}, &model.LanguagePattern{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	dao.DB = db
	t.Cleanup(func() { dao.DB = nil })
	return db
}

// setupRouter 装配真实 service + handler 的测试路由(仅本测试关注的路径)
// memory 参数本测试未涉及,传 nil
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())

	h := New(agent_service.New(dao.NewAgentDao(), dao.NewPillDao(), dao.NewModelDao()), nil)
	r.PUT("/api/v1/agents/:uuid/pills/:pill_uuid", router.Wrapper(h.UpdateAgentPill))
	return r
}

// seedBoundPair 造数: 道人 + 金丹 + 服用记录,返回双方 UUID
func seedBoundPair(t *testing.T, db *gorm.DB) (string, string) {
	t.Helper()

	agent := model.DaoAgent{Name: "测试道人", ModelName: "gpt-4o", Status: "active"}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("创建测试道人失败: %v", err)
	}
	pill := model.ElixirPill{Name: "测试金丹", SkillSchema: model.JSONMap{"expression_dna": map[string]interface{}{"rhythm": "快"}}}
	if err := db.Create(&pill).Error; err != nil {
		t.Fatalf("创建测试金丹失败: %v", err)
	}
	if err := db.Create(&model.AgentPill{AgentID: agent.ID, PillID: pill.ID, Weight: 1.0, SortOrder: 1}).Error; err != nil {
		t.Fatalf("创建测试服用记录失败: %v", err)
	}
	return agent.UUID.String(), pill.UUID.String()
}

// putJSON 发送 PUT JSON 请求并解析响应包络
func putJSON(t *testing.T, r *gin.Engine, path string, body string) (int, map[string]interface{}) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应包络失败: %v, body: %s", err, w.Body.String())
	}
	return w.Code, envelope
}

// TestUpdateAgentPill_NoPillIDInBody 修复验证: body 仅 {weight, sort_order} 即可成功(原 bug 要求 pill_id 报 400)
func TestUpdateAgentPill_NoPillIDInBody(t *testing.T) {
	db := setupTestDB(t)
	agentUUID, pillUUID := seedBoundPair(t, db)
	r := setupRouter()

	status, envelope := putJSON(t, r,
		fmt.Sprintf("/api/v1/agents/%s/pills/%s", agentUUID, pillUUID),
		`{"weight": 2, "sort_order": 3}`)

	if status != http.StatusOK {
		t.Fatalf("期望 HTTP 200, 实际 %d, body: %v", status, envelope)
	}
	if envelope["code"].(float64) != 0 {
		t.Fatalf("期望 code=0, 实际 %v", envelope)
	}

	// 数据确实更新
	var ap model.AgentPill
	if err := db.First(&ap).Error; err != nil {
		t.Fatalf("查询服用记录失败: %v", err)
	}
	if ap.Weight != 2 || ap.SortOrder != 3 {
		t.Fatalf("期望 weight=2 sort_order=3, 实际 weight=%v sort_order=%v", ap.Weight, ap.SortOrder)
	}
}

// TestUpdateAgentPill_WeightOutOfRange 权重超上限(>10)返回 400
func TestUpdateAgentPill_WeightOutOfRange(t *testing.T) {
	db := setupTestDB(t)
	agentUUID, pillUUID := seedBoundPair(t, db)
	r := setupRouter()

	status, envelope := putJSON(t, r,
		fmt.Sprintf("/api/v1/agents/%s/pills/%s", agentUUID, pillUUID),
		`{"weight": 11, "sort_order": 3}`)

	if status != http.StatusBadRequest {
		t.Fatalf("期望 HTTP 400, 实际 %d, body: %v", status, envelope)
	}
	if envelope["code"].(float64) != 400 {
		t.Fatalf("期望 code=400, 实际 %v", envelope)
	}
}

// TestUpdateAgentPill_NonUUIDPath 路径段非 UUID 返回 400
func TestUpdateAgentPill_NonUUIDPath(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	for _, path := range []string{
		"/api/v1/agents/not-a-uuid/pills/" + uuid.NewString(),
		"/api/v1/agents/" + uuid.NewString() + "/pills/5",
	} {
		status, envelope := putJSON(t, r, path, `{"weight": 2, "sort_order": 3}`)
		if status != http.StatusBadRequest {
			t.Fatalf("路径 %s 期望 HTTP 400, 实际 %d, body: %v", path, status, envelope)
		}
	}
}

// TestUpdateAgentPill_UnknownUUID 合法 UUID 但不存在返回 404
func TestUpdateAgentPill_UnknownUUID(t *testing.T) {
	db := setupTestDB(t)
	agentUUID, _ := seedBoundPair(t, db)
	r := setupRouter()

	// 金丹 UUID 不存在
	status, envelope := putJSON(t, r,
		fmt.Sprintf("/api/v1/agents/%s/pills/%s", agentUUID, uuid.NewString()),
		`{"weight": 2, "sort_order": 3}`)
	if status != http.StatusNotFound {
		t.Fatalf("期望 HTTP 404, 实际 %d, body: %v", status, envelope)
	}

	// 道人 UUID 不存在
	status, envelope = putJSON(t, r,
		fmt.Sprintf("/api/v1/agents/%s/pills/%s", uuid.NewString(), uuid.NewString()),
		`{"weight": 2, "sort_order": 3}`)
	if status != http.StatusNotFound {
		t.Fatalf("期望 HTTP 404, 实际 %d, body: %v", status, envelope)
	}
}
