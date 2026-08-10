// Package system 系统管理 HTTP 处理器(新网关;对齐 Luna-CY 模板 handler 分包风格)
// 路由: /api/v1/system/health, /api/v1/system/config
// 系统域无业务逻辑(读配置 + ping 引擎/DB),无 service 层,无需 wire 装配,路由处内联构造
package system

// System 系统处理器
type System struct {
	version string
}

// New 构造系统处理器(version 硬编码;无 service 依赖)
func New() *System {
	return &System{version: "2.0.0"}
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

// ConfigResponse 系统配置响应 DTO(前端可见模型清单)
type ConfigResponse struct {
	Version        string   `json:"version"`         // 版本号
	Models         []string `json:"models"`          // 可用模型列表
	DefaultModel   string   `json:"default_model"`   // 默认对话模型
	SynthesisModel string   `json:"synthesis_model"` // 语言模式合成模型
}
