/**
 * API 基础封装层
 * 基于 fetch 的统一请求封装，包含错误处理、拦截器等功能
 * 前端演示模式: 使用 MOCK_DELAY 模拟网络延迟
 */

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api/v1'

// 模拟网络延迟（演示模式）
const MOCK_DELAY = 400

/**
 * 模拟延迟（演示模式使用）
 */
export function mockDelay(ms = MOCK_DELAY): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * 统一 API 响应格式
 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/**
 * 请求选项
 */
interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean>
}

/**
 * 构建完整 URL（带查询参数）
 */
function buildUrl(path: string, params?: Record<string, string | number | boolean>): string {
  const url = new URL(API_BASE + path, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, String(value))
    })
  }
  return url.toString()
}

/**
 * 统一请求函数
 * @param path API 路径
 * @param options 请求选项
 * @returns 解析后的 JSON 数据
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { params, ...fetchOptions } = options
  const url = buildUrl(path, params)

  // 默认请求头
  const headers: Record<string, string> = {
    'Accept': 'application/json',
    ...(!(fetchOptions.body instanceof FormData) && { 'Content-Type': 'application/json' }),
    ...((fetchOptions.headers as Record<string, string>) || {}),
  }

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      headers,
    })

    // 处理 HTTP 错误
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new ApiError(
        errorData.message || `HTTP ${response.status}: ${response.statusText}`,
        response.status,
        errorData
      )
    }

    // 204 No Content
    if (response.status === 204) {
      return undefined as T
    }

    return await response.json()
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    // 网络错误
    throw new ApiError(
      error instanceof Error ? error.message : '网络请求失败，请检查网络连接',
      0
    )
  }
}

/**
 * 自定义 API 错误类
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: Record<string, unknown>
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/**
 * GET 请求快捷方法
 */
export function get<T>(path: string, params?: Record<string, string | number | boolean>): Promise<T> {
  return request<T>(path, { method: 'GET', params })
}

/**
 * POST 请求快捷方法
 */
export function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    body: body instanceof FormData ? body : JSON.stringify(body),
  })
}

/**
 * PUT 请求快捷方法
 */
export function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

/**
 * DELETE 请求快捷方法
 */
export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}

/**
 * 上传文件请求
 * @param path API 路径
 * @param files 文件列表
 * @param extraData 额外表单数据
 * @returns 解析后的 JSON 数据
 */
export function upload<T>(
  path: string,
  files: File[],
  extraData?: Record<string, string>
): Promise<T> {
  const formData = new FormData()
  files.forEach(file => formData.append('files[]', file))
  if (extraData) {
    Object.entries(extraData).forEach(([key, value]) => {
      formData.append(key, value)
    })
  }
  return request<T>(path, {
    method: 'POST',
    body: formData,
  })
}
