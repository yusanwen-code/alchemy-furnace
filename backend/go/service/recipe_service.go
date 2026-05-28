// Package service 丹方业务逻辑层
// 处理丹方文件的上传、查询、删除和重新提取
// 丹方是用户上传的文档文件，经 Python RAG 提取文本后向量化入库
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	"github.com/alchemy-furnace/server/pkg/utils"
	"go.uber.org/zap"
)

// RecipeService 丹方业务逻辑
type RecipeService struct {
	ragBaseURL string
	uploadDir  string
	maxSize    int64 // MB
}

// NewRecipeService 创建丹方业务实例
func NewRecipeService() *RecipeService {
	cfg := config.Get()
	return &RecipeService{
		ragBaseURL: cfg.PythonRAG.BaseURL,
		uploadDir:  cfg.Upload.Dir,
		maxSize:    cfg.Upload.MaxSize,
	}
}

// ListRecipesByPill 获取指定金丹下的丹方列表
func (s *RecipeService) ListRecipesByPill(pillID uint, page, pageSize int) ([]model.ElixirRecipe, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var recipes []model.ElixirRecipe
	var total int64

	db := dao.GetDB().Model(&model.ElixirRecipe{}).Where("pill_id = ?", pillID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询丹方总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&recipes).Error; err != nil {
		return nil, 0, fmt.Errorf("查询丹方列表失败: %w", err)
	}

	return recipes, total, nil
}

// UploadRecipes 批量上传丹方文件到指定金丹
// 步骤: 1. 校验文件 2. 保存到磁盘 3. 记录数据库 4. 异步调用 RAG 提取和向量化
func (s *RecipeService) UploadRecipes(pillID uint, files []*multipart.FileHeader) ([]*model.ElixirRecipe, error) {
	// 1. 检查金丹是否存在
	var pill model.ElixirPill
	if err := dao.GetDB().First(&pill, pillID).Error; err != nil {
		return nil, fmt.Errorf("金丹(id=%d)不存在: %w", pillID, err)
	}

	// 2. 确保上传目录存在
	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	var recipes []*model.ElixirRecipe
	for _, fileHeader := range files {
		// 3. 校验文件大小（限制 100MB）
		if fileHeader.Size > s.maxSize*1024*1024 {
			return nil, fmt.Errorf("文件 %s 过大: %s, 最大允许 %dMB",
				fileHeader.Filename, utils.FormatFileSize(fileHeader.Size), s.maxSize)
		}

		// 4. 校验文件类型
		if !utils.IsAllowedFileType(fileHeader.Filename) {
			return nil, fmt.Errorf("不支持的文件类型: %s", fileHeader.Filename)
		}

		// 5. 生成唯一文件名并保存
		uniqueName := utils.GenerateUniqueFilename(fileHeader.Filename)
		savePath := filepath.Join(s.uploadDir, uniqueName)

		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("打开上传文件失败: %w", err)
		}

		dst, err := os.Create(savePath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("创建目标文件失败: %w", err)
		}

		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			os.Remove(savePath)
			return nil, fmt.Errorf("保存文件失败: %w", err)
		}
		src.Close()
		dst.Close()

		// 6. 创建数据库记录
		recipe := model.ElixirRecipe{
			PillID:        pillID,
			Filename:      fileHeader.Filename,
			FileType:      utils.GetFileType(fileHeader.Filename),
			FileSize:      fileHeader.Size,
			FilePath:      savePath,
			ExtractStatus: "pending",
			ChunkCount:    0,
		}

		if err := dao.GetDB().Create(&recipe).Error; err != nil {
			os.Remove(savePath) // 清理已保存的文件
			return nil, fmt.Errorf("保存丹方记录失败: %w", err)
		}

		recipes = append(recipes, &recipe)
		zap.L().Info("[炼丹炉] 丹方已收录",
			zap.String("filename", recipe.Filename),
			zap.String("type", recipe.FileType),
			zap.Int64("size", recipe.FileSize))

		// 7. 异步调用 Python RAG 进行提取和向量化（不阻塞上传响应）
		go s.asyncExtractAndVectorize(&recipe)
	}

	return recipes, nil
}

// DeleteRecipe 删除丹方，同时删除物理文件和向量数据
func (s *RecipeService) DeleteRecipe(id uint) error {
	var recipe model.ElixirRecipe
	if err := dao.GetDB().First(&recipe, id).Error; err != nil {
		return fmt.Errorf("丹方(id=%d)不存在: %w", id, err)
	}

	// 删除数据库记录
	if err := dao.GetDB().Delete(&recipe).Error; err != nil {
		return fmt.Errorf("删除丹方记录失败: %w", err)
	}

	// 删除物理文件（异步，不阻塞）
	if recipe.FilePath != "" {
		go func(path string) {
			if err := os.Remove(path); err != nil {
				zap.L().Warn("[炼丹炉] 删除文件失败", zap.String("path", path), zap.Error(err))
			}
		}(recipe.FilePath)
	}

	// 更新金丹的向量数量（可选，通过后台任务同步）
	zap.L().Info("[炼丹炉] 丹方已删除", zap.Uint("id", id), zap.String("filename", recipe.Filename))
	return nil
}

// ReExtract 重新提取丹方内容并更新向量
func (s *RecipeService) ReExtract(id uint) error {
	var recipe model.ElixirRecipe
	if err := dao.GetDB().First(&recipe, id).Error; err != nil {
		return fmt.Errorf("丹方(id=%d)不存在: %w", id, err)
	}

	// 更新状态为提取中
	if err := dao.GetDB().Model(&recipe).Update("extract_status", "extracting").Error; err != nil {
		return fmt.Errorf("更新提取状态失败: %w", err)
	}

	// 异步重新提取
	go s.asyncExtractAndVectorize(&recipe)

	zap.L().Info("[炼丹炉] 丹方重新提取任务已下发", zap.Uint("id", id))
	return nil
}

// asyncExtractAndVectorize 异步调用 Python RAG 进行文档提取和向量化入库
// 步骤: 1. 提取文档文本 2. 文本切分 3. Embedding 4. 存入 Qdrant
func (s *RecipeService) asyncExtractAndVectorize(recipe *model.ElixirRecipe) {
	zap.L().Info("[炼丹炉] 开始异步炼丹...",
		zap.Uint("recipe_id", recipe.ID),
		zap.String("filename", recipe.Filename))

	// 1. 调用 Python RAG 提取文档内容
	extractResult, err := s.callExtractDocument(recipe)
	if err != nil {
		zap.L().Error("[炼丹炉] 文档提取失败",
			zap.Uint("recipe_id", recipe.ID), zap.Error(err))
		s.updateExtractStatus(recipe.ID, "failed", err.Error(), 0)
		return
	}

	// 2. 调用 Python RAG 进行向量化入库
	if err := s.callIngestVectors(recipe, extractResult); err != nil {
		zap.L().Error("[炼丹炉] 向量化入库失败",
			zap.Uint("recipe_id", recipe.ID), zap.Error(err))
		s.updateExtractStatus(recipe.ID, "failed", err.Error(), 0)
		return
	}

	// 3. 更新成功状态
	chunkCount := 0
	if chunks, ok := extractResult["chunks"].([]interface{}); ok {
		chunkCount = len(chunks)
	}
	s.updateExtractStatus(recipe.ID, "success", "提取和向量化成功", chunkCount)

	// 4. 更新金丹状态
	s.updatePillStatus(recipe.PillID)

	zap.L().Info("[炼丹炉] 炼丹完成！丹方已转化为金丹之力",
		zap.Uint("recipe_id", recipe.ID),
		zap.Int("chunks", chunkCount))
}

// callExtractDocument 调用 Python RAG 的文档提取接口
func (s *RecipeService) callExtractDocument(recipe *model.ElixirRecipe) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/documents/extract", s.ragBaseURL)

	reqBody := map[string]interface{}{
		"file_path": recipe.FilePath,
		"file_type": recipe.FileType,
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 120 * time.Second} // 大文件提取可能需要较长时间
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("调用文档提取接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("文档提取接口返回错误: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析提取结果失败: %w", err)
	}

	return result, nil
}

// callIngestVectors 调用 Python RAG 的向量化入库接口
func (s *RecipeService) callIngestVectors(recipe *model.ElixirRecipe, extractResult map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/v1/vectors/ingest", s.ragBaseURL)

	// 构建文本和 chunks
	text := ""
	if t, ok := extractResult["text"].(string); ok {
		text = t
	}

	reqBody := map[string]interface{}{
		"pill_id":   recipe.PillID,
		"recipe_id": recipe.ID,
		"text":      text,
		"chunks":    extractResult["chunks"],
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("调用向量化入库接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("向量化入库接口返回错误: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// updateExtractStatus 更新丹方的提取状态和结果
func (s *RecipeService) updateExtractStatus(recipeID uint, status, result string, chunkCount int) {
	updates := map[string]interface{}{
		"extract_status": status,
		"extract_result": result,
		"chunk_count":    chunkCount,
	}
	if err := dao.GetDB().Model(&model.ElixirRecipe{}).Where("id = ?", recipeID).Updates(updates).Error; err != nil {
		zap.L().Error("[炼丹炉] 更新提取状态失败", zap.Uint("recipe_id", recipeID), zap.Error(err))
	}
}

// updatePillStatus 更新金丹的炼丹状态
// 当一个金丹下所有丹方都提取成功后，将金丹状态更新为 refined
func (s *RecipeService) updatePillStatus(pillID uint) {
	var pendingCount int64
	dao.GetDB().Model(&model.ElixirRecipe{}).
		Where("pill_id = ? AND extract_status IN (?)", pillID, []string{"pending", "extracting"}).
		Count(&pendingCount)

	if pendingCount == 0 {
		// 所有丹方都已处理完毕，检查是否有成功的
		var successCount int64
		dao.GetDB().Model(&model.ElixirRecipe{}).
			Where("pill_id = ? AND extract_status = ?", pillID, "success").
			Count(&successCount)

		status := "failed"
		if successCount > 0 {
			status = "refined"
		}

		if err := dao.GetDB().Model(&model.ElixirPill{}).Where("id = ?", pillID).
			Update("status", status).Error; err != nil {
			zap.L().Error("[炼丹炉] 更新金丹状态失败", zap.Uint("pill_id", pillID), zap.Error(err))
		} else {
			zap.L().Info("[炼丹炉] 金丹状态更新", zap.Uint("pill_id", pillID), zap.String("status", status))
		}
	}
}

// GetRecipe 根据 ID 获取丹方详情
func (s *RecipeService) GetRecipe(id uint) (*model.ElixirRecipe, error) {
	var recipe model.ElixirRecipe
	if err := dao.GetDB().First(&recipe, id).Error; err != nil {
		return nil, fmt.Errorf("丹方(id=%d)不存在: %w", id, err)
	}
	return &recipe, nil
}

// DeleteFile 删除物理文件（辅助函数）
func (s *RecipeService) DeleteFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}
