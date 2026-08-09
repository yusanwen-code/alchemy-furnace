package system

import (
	"net/http"
	"time"

	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
// GET /api/v1/system/health
// 返回各组件健康状态,用于监控与负载均衡
// ⚠️ 新架构经 Wrapper 写出统一响应包络 {code,message,request_id,data};
// 旧实现直接 c.JSON 绕过包络(无 request_id),此处修复为 return (Ok, data, nil)
func (cls *System) HealthCheck(c *gin.Context) (response.Code, any, error) {
	dbStatus := "ok"
	if sqlDB, err := dao.GetDB().DB(); err != nil || sqlDB.Ping() != nil {
		dbStatus = "down"
	}

	pythonEngine := "down"
	if pingEngine(configuration.Configuration.PythonEngine.BaseURL) {
		pythonEngine = "ok"
	}

	status := "ok"
	if dbStatus != "ok" || pythonEngine != "ok" {
		status = "degraded"
	}

	return response.Ok, &HealthResponse{
		Status:       status,
		Version:      cls.version,
		Timestamp:    time.Now().Unix(),
		DB:           dbStatus,
		PythonEngine: pythonEngine,
	}, nil
}

// pingEngine 检测 Python 语言引擎连通性: GET baseURL+/health,3s 超时,200 视为可达
func pingEngine(baseURL string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
