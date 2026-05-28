// Package model 定义「炼丹炉」的全部 GORM 数据模型
// 对应数据库表：金丹(elixir_pills)、丹方(elixir_recipes)、道人(dao_agents)、
// 服用记录(agent_pills)、对话会话(chat_sessions)、对话消息(chat_messages)
// 所有模型使用 GORM v2 标签，支持自动迁移
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ---------- 金丹（知识库） ----------

// ElixirPill 金丹模型，对应 elixir_pills 表
// 金丹是知识库的载体，一个金丹包含多个丹方（文档），对应 Qdrant 中一组向量
type ElixirPill struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement;comment:金丹唯一标识"`
	Name        string    `json:"name" gorm:"size:100;not null;comment:金丹名称"`
	Description string    `json:"description" gorm:"type:text;comment:金丹描述"`
	Status      string    `json:"status" gorm:"size:20;default:refining;comment:炼丹状态: refining(炼制中)/refined(炼制成功)/failed(炼制失败)"`
	VectorCount int       `json:"vector_count" gorm:"default:0;comment:向量数量"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	// 关联关系：一个金丹有多个丹方
	Recipes []ElixirRecipe `json:"recipes,omitempty" gorm:"foreignKey:PillID;references:ID;constraint:OnDelete:CASCADE;"`
	// 关联关系：一个金丹被多个道人服用
	AgentPills []AgentPill `json:"agent_pills,omitempty" gorm:"foreignKey:PillID;references:ID;constraint:OnDelete:CASCADE;"`
}

// TableName 指定表名
func (ElixirPill) TableName() string {
	return "elixir_pills"
}

// ---------- 丹方（文档文件） ----------

// ElixirRecipe 丹方模型，对应 elixir_recipes 表
// 丹方是用户上传的文档文件，经过 Python RAG 提取文本并切分后向量化入库
type ElixirRecipe struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement;comment:丹方唯一标识"`
	PillID        uint      `json:"pill_id" gorm:"not null;index;comment:所属金丹ID"`
	Filename      string    `json:"filename" gorm:"size:255;not null;comment:原始文件名"`
	FileType      string    `json:"file_type" gorm:"size:50;not null;comment:文件类型:doc/xlsx/md/txt/pdf/audio/video"`
	FileSize      int64     `json:"file_size" gorm:"comment:文件大小(字节)"`
	FilePath      string    `json:"file_path" gorm:"size:500;comment:文件存储路径"`
	ExtractStatus string    `json:"extract_status" gorm:"size:20;default:pending;comment:提取状态: pending(待提取)/extracting(提取中)/success(提取成功)/failed(提取失败)"`
	ExtractResult string    `json:"extract_result" gorm:"type:text;comment:提取的文本内容摘要"`
	ChunkCount    int       `json:"chunk_count" gorm:"default:0;comment:文本切分块数"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	// 关联关系：丹方属于一个金丹
	Pill ElixirPill `json:"pill,omitempty" gorm:"foreignKey:PillID;references:ID"`
}

// TableName 指定表名
func (ElixirRecipe) TableName() string {
	return "elixir_recipes"
}

// ---------- 道人（AI Agent） ----------

// DaoAgent 道人模型，对应 dao_agents 表
// 道人是 AI 对话代理，拥有独特的性格和系统提示词，可服用多个金丹获得知识
type DaoAgent struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement;comment:道人唯一标识"`
	Name        string    `json:"name" gorm:"size:100;not null;comment:道人名称"`
	Avatar      string    `json:"avatar" gorm:"size:255;comment:头像URL"`
	Personality string    `json:"personality" gorm:"type:text;comment:性格描述/系统提示词"`
	ModelName   string    `json:"model_name" gorm:"size:50;default:gpt-4o;comment:使用的LLM模型名称"`
	Status      string    `json:"status" gorm:"size:20;default:active;comment:状态: active(活跃)/inactive(停用)"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	// 关联关系：一个道人服用多个金丹
	AgentPills []AgentPill `json:"agent_pills,omitempty" gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE;"`
	// 关联关系：一个道人参与多个会话
	Sessions []ChatSession `json:"sessions,omitempty" gorm:"foreignKey:AgentID;references:ID;constraint:OnDelete:CASCADE;"`
}

// TableName 指定表名
func (DaoAgent) TableName() string {
	return "dao_agents"
}

// ---------- 服用记录（Agent 绑定金丹） ----------

// AgentPill 服用记录模型，对应 agent_pills 表
// 记录道人与金丹的绑定关系，一个道人可以服用多个金丹，一个金丹可被多个道人服用
// agent_id 和 pill_id 联合唯一
type AgentPill struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement;comment:服用记录唯一标识"`
	AgentID   uint      `json:"agent_id" gorm:"not null;uniqueIndex:idx_agent_pill;comment:道人ID"`
	PillID    uint      `json:"pill_id" gorm:"not null;uniqueIndex:idx_agent_pill;comment:金丹ID"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;comment:服用时间"`

	// 关联关系
	Agent DaoAgent   `json:"agent,omitempty" gorm:"foreignKey:AgentID;references:ID"`
	Pill  ElixirPill `json:"pill,omitempty" gorm:"foreignKey:PillID;references:ID"`
}

// TableName 指定表名
func (AgentPill) TableName() string {
	return "agent_pills"
}

// ---------- 对话会话 ----------

// ChatSession 对话会话模型，对应 chat_sessions 表
// 用户与某个道人之间的对话上下文，一个会话包含多条消息
type ChatSession struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement;comment:会话唯一标识"`
	AgentID   uint      `json:"agent_id" gorm:"not null;index;comment:关联的道人ID"`
	Title     string    `json:"title" gorm:"size:200;comment:会话标题"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	// 关联关系
	Agent    DaoAgent       `json:"agent,omitempty" gorm:"foreignKey:AgentID;references:ID"`
	Messages []ChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE;"`
}

// TableName 指定表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ---------- 对话消息 ----------

// ChatMessage 对话消息模型，对应 chat_messages 表
// 存储用户与道人的对话内容，包括 RAG 引用来源
// role: user(用户提问) / assistant(道人回答) / system(系统提示)
type ChatMessage struct {
	ID        uint        `json:"id" gorm:"primaryKey;autoIncrement;comment:消息唯一标识"`
	SessionID uint        `json:"session_id" gorm:"not null;index;comment:所属会话ID"`
	Role      string      `json:"role" gorm:"size:20;not null;comment:角色: user/assistant/system"`
	Content   string      `json:"content" gorm:"type:text;not null;comment:消息内容"`
	Sources   JSONMap     `json:"sources" gorm:"type:jsonb;comment:RAG引用来源(JSONB格式)"`
	CreatedAt time.Time   `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	// 关联关系
	Session ChatSession `json:"session,omitempty" gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// ---------- JSONB 支持 ----------

// JSONMap 是 map[string]interface{} 的包装类型，用于支持 PostgreSQL 的 JSONB 字段
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口，将 JSONMap 转换为 JSON 字符串存入数据库
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

// Scan 实现 sql.Scanner 接口，从数据库 JSON 字符串扫描为 JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("不支持的 JSONB 扫描类型")
	}
	if len(bytes) == 0 {
		*j = JSONMap{}
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// ---------- 请求/响应 DTO ----------

// CreatePillRequest 创建金丹请求
 type CreatePillRequest struct {
	Name        string `json:"name" binding:"required,max=100"` // 金丹名称
	Description string `json:"description"`                    // 金丹描述
}

// UpdatePillRequest 更新金丹请求
 type UpdatePillRequest struct {
	Name        string `json:"name" binding:"max=100"` // 金丹名称
	Description string `json:"description"`             // 金丹描述
}

// CreateAgentRequest 创建道人请求
 type CreateAgentRequest struct {
	Name        string `json:"name" binding:"required,max=100"`       // 道人名称
	Avatar      string `json:"avatar"`                                  // 头像URL
	Personality string `json:"personality"`                             // 性格描述/系统提示词
	ModelName   string `json:"model_name" binding:"max=50"`             // 使用的LLM模型
}

// UpdateAgentRequest 更新道人请求
 type UpdateAgentRequest struct {
	Name        string `json:"name" binding:"max=100"`      // 道人名称
	Avatar      string `json:"avatar"`                       // 头像URL
	Personality string `json:"personality"`                  // 性格描述/系统提示词
	ModelName   string `json:"model_name" binding:"max=50"`  // 使用的LLM模型
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"` // 状态
}

// BindPillRequest 服用金丹请求
 type BindPillRequest struct {
	PillID uint `json:"pill_id" binding:"required"` // 金丹ID
}

// CreateSessionRequest 创建会话请求
 type CreateSessionRequest struct {
	AgentID uint   `json:"agent_id" binding:"required"` // 道人ID
	Title   string `json:"title" binding:"max=200"`     // 会话标题
}

// ChatMessageRequest 聊天消息请求结构
 type ChatMessageRequest struct {
	Content string `json:"content" binding:"required"` // 消息内容
}

// HealthCheckResponse 健康检查响应
 type HealthCheckResponse struct {
	Status    string `json:"status"`              // 状态: ok/degraded/down
	Version   string `json:"version"`             // 版本号
	Timestamp int64  `json:"timestamp"`           // 时间戳
	DB        string `json:"db"`                  // 数据库状态
	Qdrant    string `json:"qdrant"`              // Qdrant状态
	PythonRAG string `json:"python_rag"`          // Python RAG服务状态
}
