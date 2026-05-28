// Package response 提供「炼丹炉」统一的 HTTP 响应格式
// 所有接口返回统一结构: { "code": 0, "message": "ok", "data": {} }
// code = 0 表示成功，非零表示各种错误，使用 4xx/5xx HTTP 状态码辅助语义表达
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构体，所有 API 接口都返回此格式
type Response struct {
	Code    int         `json:"code"`    // 业务状态码: 0=成功, 非0=各种错误
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 响应数据
}

// PageInfo 分页信息结构体，用于列表接口
type PageInfo struct {
	Total    int64       `json:"total"`     // 总记录数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页大小
	List     interface{} `json:"list"`      // 列表数据
}

// ---------- 成功响应 ----------

// Success 返回成功响应，data 为具体业务数据
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// SuccessWithPage 返回分页成功响应，用于列表查询
func SuccessWithPage(c *gin.Context, total int64, page, pageSize int, list interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data: PageInfo{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			List:     list,
		},
	})
}

// Created 返回创建成功响应（HTTP 201）
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "创建成功",
		Data:    data,
	})
}

// NoContent 返回无内容成功响应（HTTP 204），用于删除操作
func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, nil)
}

// ---------- 错误响应 ----------

// Error 返回通用错误响应，传入业务错误码和错误信息
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorWithStatus 返回带 HTTP 状态码的错误响应
func ErrorWithStatus(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// BadRequest 返回 400 错误（请求参数错误）
func BadRequest(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusBadRequest, 400, message)
}

// Unauthorized 返回 401 错误（未认证）
func Unauthorized(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusUnauthorized, 401, message)
}

// Forbidden 返回 403 错误（无权限）
func Forbidden(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusForbidden, 403, message)
}

// NotFound 返回 404 错误（资源不存在）
func NotFound(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusNotFound, 404, message)
}

// InternalError 返回 500 错误（服务器内部错误）
func InternalError(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusInternalServerError, 500, message)
}

// ---------- 常用错误码 ----------

const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// CodeBadRequest 请求参数错误
	CodeBadRequest = 400

	// CodeUnauthorized 未认证
	CodeUnauthorized = 401

	// CodeForbidden 无权限
	CodeForbidden = 403

	// CodeNotFound 资源不存在
	CodeNotFound = 404

	// CodeInternalError 服务器内部错误
	CodeInternalError = 500

	// CodeDBError 数据库操作失败
	CodeDBError = 1001

	// CodeRAGError Python RAG 服务调用失败
	CodeRAGError = 2001

	// CodeFileUploadError 文件上传失败
	CodeFileUploadError = 3001

	// CodeFileTypeNotAllowed 不支持的文件类型
	CodeFileTypeNotAllowed = 3002

	// CodeFileTooLarge 文件过大
	CodeFileTooLarge = 3003

	// CodeWebSocketError WebSocket 通信错误
	CodeWebSocketError = 4001
)
