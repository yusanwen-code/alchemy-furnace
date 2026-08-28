import { ApiError, authHeaders, buildApiUrl, post } from './api'
import type { DistillationDraft, SkillExportRequest, SkillExportResult } from './types'

export function distillNuwa(input: {
  subject: string
  brief: string
  locale: 'zh-CN' | 'en'
}): Promise<DistillationDraft> {
  return post<DistillationDraft>('/distillation/nuwa', input)
}

/** 从 Content-Disposition 解析纯 ASCII 下载名;缺失或无法解析时回退通用名 */
function parseExportFilename(contentDisposition: string | null): string {
  const match = contentDisposition?.match(/filename="?([^";]+)"?/)
  return match?.[1] || 'alchemy-skill-export.zip'
}

/**
 * 请求 Skill 导出 ZIP(二进制响应,不走 JSON 信封)。
 * 下载失败抛 ApiError(携带稳定 error_code;503 可重试错误由调用方展示重试入口)。
 */
export async function exportSkill(input: SkillExportRequest): Promise<SkillExportResult> {
  let response: Response
  try {
    response = await fetch(buildApiUrl('/distillation/skill-export'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders(),
      },
      body: JSON.stringify(input),
    })
  } catch (error) {
    throw new ApiError(
      error instanceof Error ? error.message : '网络请求失败，请检查网络连接',
      0
    )
  }

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new ApiError(
      (errorData as { message?: string }).message || `HTTP ${response.status}: ${response.statusText}`,
      response.status,
      errorData as Record<string, unknown>
    )
  }

  const blob = await response.blob()
  return { blob, filename: parseExportFilename(response.headers.get('Content-Disposition')) }
}

/** Blob → base64 文本(桌面桥接端点走 JSON 文本传输) */
async function blobToBase64(blob: Blob): Promise<string> {
  const bytes = new Uint8Array(await blob.arrayBuffer())
  let binary = ''
  const CHUNK = 0x8000
  for (let i = 0; i < bytes.length; i += CHUNK) {
    binary += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return btoa(binary)
}

/**
 * 桌面端保存 Skill 导出:POST 桌面桥接端点,落盘到数据目录 exports/。
 * (WKWebView 不执行 Blob a[download],桌面环境必须走此桥接;浏览器环境请保留 exportSkill 的 Blob 下载)
 * @returns 已保存文件的绝对路径
 */
export async function saveSkillExportDesktop(blob: Blob, filename: string): Promise<string> {
  const data = await post<{ saved_path: string }>('/desktop/save-export', {
    filename,
    content_b64: await blobToBase64(blob),
  })
  return data.saved_path
}

/** 桌面端在系统文件管理器中定位已保存的导出文件(仅 desktop 模式端点存在) */
export async function revealSkillExport(path: string): Promise<void> {
  await post('/desktop/reveal-export', { path })
}
