// Package service 金丹业务逻辑层
// 处理金丹的增删改查，以及级联删除相关的向量数据和丹方
// 金丹是知识库的载体，每个金丹包含多个丹方（文档）
package service

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PillService 金丹业务逻辑
type PillService struct {
	ragBaseURL string // Python RAG 服务地址
}

// NewPillService 创建金丹业务实例
func NewPillService() *PillService {
	return &PillService{
		ragBaseURL: config.Get().PythonRAG.BaseURL,
	}
}

// ListPills 获取金丹列表，支持分页和关键字搜索
func (s *PillService) ListPills(page, pageSize int, keyword string) ([]model.ElixirPill, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var pills []model.ElixirPill
	var total int64

	db := dao.GetDB().Model(&model.ElixirPill{})

	// 关键字搜索（名称或描述）
	if keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询金丹总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&pills).Error; err != nil {
		return nil, 0, fmt.Errorf("查询金丹列表失败: %w", err)
	}

	return pills, total, nil
}

// GetPill 根据 ID 获取金丹详情，包含关联的丹方列表
func (s *PillService) GetPill(id uint) (*model.ElixirPill, error) {
	var pill model.ElixirPill
	if err := dao.GetDB().Preload("Recipes").First(&pill, id).Error; err != nil {
		return nil, fmt.Errorf("查询金丹(id=%d)失败: %w", id, err)
	}
	return &pill, nil
}

// CreatePill 创建新的金丹
func (s *PillService) CreatePill(req *model.CreatePillRequest) (*model.ElixirPill, error) {
	pill := model.ElixirPill{
		Name:        req.Name,
		Description: req.Description,
		Status:      "refining",
		VectorCount: 0,
	}

	if err := dao.GetDB().Create(&pill).Error; err != nil {
		return nil, fmt.Errorf("创建金丹失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 金丹炼制启动", zap.String("name", pill.Name), zap.Uint("id", pill.ID))
	return &pill, nil
}

// UpdatePill 更新金丹信息
func (s *PillService) UpdatePill(id uint, req *model.UpdatePillRequest) (*model.ElixirPill, error) {
	var pill model.ElixirPill
	if err := dao.GetDB().First(&pill, id).Error; err != nil {
		return nil, fmt.Errorf("金丹(id=%d)不存在: %w", id, err)
	}

	// 只更新非空字段
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := dao.GetDB().Model(&pill).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新金丹(id=%d)失败: %w", id, err)
	}

	// 重新查询获取更新后的数据
	if err := dao.GetDB().First(&pill, id).Error; err != nil {
		return nil, fmt.Errorf("查询更新后的金丹失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 金丹信息已更新", zap.Uint("id", pill.ID), zap.String("name", pill.Name))
	return &pill, nil
}

// DeletePill 删除金丹，级联删除所有关联的丹方文件和向量数据
// 这是一个危险操作，会：1. 删除 Qdrant 中的向量 2. 删除数据库记录 3. 删除上传的文件
func (s *PillService) DeletePill(id uint) error {
	db := dao.GetDB()

	// 1. 先获取金丹信息，确认存在
	var pill model.ElixirPill
	if err := db.First(&pill, id).Error; err != nil {
		return fmt.Errorf("金丹(id=%d)不存在: %w", id, err)
	}

	// 2. 获取关联的丹方列表（用于后续删除文件）
	var recipes []model.ElixirRecipe
	if err := db.Where("pill_id = ?", id).Find(&recipes).Error; err != nil {
		zap.L().Warn("[炼丹炉] 查询关联丹方失败", zap.Uint("pill_id", id), zap.Error(err))
	}

	// 3. 删除 Qdrant 中的向量数据（调用 Python RAG 服务）
	if err := s.deleteVectorsFromRAG(id); err != nil {
		zap.L().Warn("[炼丹炉] 删除向量数据失败", zap.Uint("pill_id", id), zap.Error(err))
		// 不阻断删除流程，继续删除数据库记录
	}

	// 4. 在事务中删除数据库记录（金丹、丹方、服用记录）
	if err := dao.Transaction(func(tx *gorm.DB) error {
		// 删除关联的丹方记录
		if err := tx.Where("pill_id = ?", id).Delete(&model.ElixirRecipe{}).Error; err != nil {
			return fmt.Errorf("删除丹方记录失败: %w", err)
		}

		// 删除服用记录
		if err := tx.Where("pill_id = ?", id).Delete(&model.AgentPill{}).Error; err != nil {
			return fmt.Errorf("删除服用记录失败: %w", err)
		}

		// 删除金丹本身
		if err := tx.Delete(&pill).Error; err != nil {
			return fmt.Errorf("删除金丹失败: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	// 5. 删除物理文件
	for _, recipe := range recipes {
		if recipe.FilePath != "" {
			// 异步删除文件，不阻塞响应
			go func(path string) {
				if err := os.Remove(path); err != nil {
					zap.L().Warn("[炼丹炉] 删除文件失败", zap.String("path", path), zap.Error(err))
				}
			}(recipe.FilePath)
		}
	}

	zap.L().Info("[炼丹炉] 金丹已销毁", zap.Uint("id", id), zap.String("name", pill.Name))
	return nil
}

// deleteVectorsFromRAG 调用 Python RAG 服务删除金丹的所有向量数据
func (s *PillService) deleteVectorsFromRAG(pillID uint) error {
	url := fmt.Sprintf("%s/api/v1/vectors/pill/%d", s.ragBaseURL, pillID)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("构建删除向量请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("调用 RAG 服务删除向量失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("RAG 服务返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
