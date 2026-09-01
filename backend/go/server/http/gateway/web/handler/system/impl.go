// Package system 系统管理 HTTP 处理器(新网关;对齐 Luna-CY 模板 handler 分包风格)
// 路由: /api/v1/system/health, /api/v1/system/config
// 系统域无业务逻辑(读配置 + ping 引擎/DB),但 GetConfig 需要从 DB 读实际配置的融合模型,
// 所以注入 model service(只读,无状态变更)
package system

import (
	"path/filepath"
	"time"

	"github.com/alchemy-furnace/server/internal/engineendpoint"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/logger"
	"github.com/alchemy-furnace/server/internal/paths"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// System 系统处理器
type System struct {
	version string
	model   service.Model
}

// New 构造系统处理器
func New(model service.Model) *System {
	return &System{version: "2.0.0", model: model}
}

// ---------- 响应 DTO ----------

// HealthResponse 健康检查响应 DTO
type HealthResponse struct {
	Status       string `json:"status"`        // 状态: ok/degraded
	Version      string `json:"version"`       // 版本号
	Timestamp    int64  `json:"timestamp"`     // 时间戳(unix 秒)
	DB           string `json:"db"`            // 数据库状态: ok/down
	PythonEngine string `json:"python_engine"` // Python 语言引擎状态: ok/down
}

// ConfigResponse 系统配置响应 DTO(前端可见模型清单 + 实际配置的融合模型)
type ConfigResponse struct {
	Version         string                     `json:"version"`           // 版本号
	Models          []string                   `json:"models"`            // 可用模型列表
	DefaultModel    string                     `json:"default_model"`     // 默认对话模型(env 兜底名)
	SynthesisModel  string                     `json:"synthesis_model"`   // 语言模式合成模型(env 兜底名)
	FusionModel     string                     `json:"fusion_model"`      // 金丹融合专用模型(env 兜底名,无 is_fusion 时使用)
	FusionModelInfo *service.FusionModelConfig `json:"fusion_model_info"` // 实际配置的融合模型(供 /fusion banner 展示)
}

// DiagnosticsResponse contains only safe local troubleshooting metadata.
type DiagnosticsResponse struct {
	Timestamp    int64  `json:"timestamp"`
	LogDir       string `json:"log_dir"`
	AppLog       string `json:"app_log"`
	PythonLog    string `json:"python_log"`
	PythonEngine string `json:"python_engine"`
}

func (cls *System) Diagnostics(_ *gin.Context) (response.Code, any, error) {
	dir, err := paths.DataDir()
	if err != nil {
		return response.ServerInternalError, nil, err
	}
	pythonEngine := "down"
	if pingEngine(engineendpoint.Current()) {
		pythonEngine = "ok"
	}
	return response.Ok, &DiagnosticsResponse{
		Timestamp: time.Now().Unix(), LogDir: logger.LogDir(dir),
		AppLog:       filepath.Join(logger.LogDir(dir), "app.log"),
		PythonLog:    filepath.Join(logger.LogDir(dir), "python.log"),
		PythonEngine: pythonEngine,
	}, nil
}
