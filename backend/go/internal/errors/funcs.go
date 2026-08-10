// 常用错误构造函数(定位码调用方传入,硬编码在发生错误的地方)
package errors

// ErrorRecordNotFound 记录不存在(->404)
func ErrorRecordNotFound(code string) Error {
	return New(ErrorTypeRecordNotFound, code, "数据不存在")
}

// ErrorInvalidRequest 请求参数错误(->400)
func ErrorInvalidRequest(code string) Error {
	return New(ErrorTypeInvalidRequest, code, "参数错误")
}

// ErrorConflict 资源冲突(->409)
func ErrorConflict(code string, message string, values ...any) Error {
	return New(ErrorTypeConflict, code, message, values...)
}

// ErrorConflictWithData 资源冲突并携带附加响应数据(->409,data 写入响应体)
func ErrorConflictWithData(code string, data any, message string, values ...any) Error {
	return NewWithData(ErrorTypeConflict, code, data, message, values...)
}

// ErrorServerInternalError 服务器内部错误(->500)
func ErrorServerInternalError(code string) Error {
	return New(ErrorTypeServerInternalError, code, "服务器内部错误")
}
