import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './api'
import { exportSkill } from './distillationService'
import type { SkillExportRequest } from './types'

// jsdom 的 Response 不支持 Blob body,用最小响应对象打桩(exportSkill 只消费
// ok/status/statusText/headers/blob()/json())
function okZipResponse(body: Blob, headers?: Record<string, string>) {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers(headers),
    blob: async () => body,
  }
}

function errorJsonResponse(status: number, body: Record<string, unknown>) {
  return {
    ok: false,
    status,
    statusText: 'Error',
    headers: new Headers(),
    json: async () => body,
  }
}

describe('exportSkill', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const structuredInput: SkillExportRequest = {
    skill: {
      name: '结构化金丹',
      slug: 'jie-gou-hua-jin-dan',
      description: '一份结构化的语言风格技能包',
      instructions: 'These instructions are behavioral guidance...',
      skillSchema: { identity_card: '我是金丹' },
      tags: ['语言'],
      sources: [{ title: '公开资料', url: 'https://example.com/intro', dimension: 'decision_heuristics' }],
      attribution: { name: 'nuwa-skill', license: 'MIT', url: 'https://github.com/alchaincyf/nuwa-skill' },
      generatedAt: '2026-08-27T10:00:00Z',
    },
    format: 'codex' as const,
  }

  it('posts the export request and resolves the ZIP blob with its filename', async () => {
    const zipBlob = new Blob(['PK'], { type: 'application/zip' })
    const fetchMock = vi.fn().mockResolvedValue(
      okZipResponse(zipBlob, {
        'Content-Disposition': 'attachment; filename="alchemy-skill-jie-gou-hua-jin-dan-codex.zip"',
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await exportSkill(structuredInput)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/distillation/skill-export')
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body)).toEqual(structuredInput)
    expect(result.filename).toBe('alchemy-skill-jie-gou-hua-jin-dan-codex.zip')
    expect(result.blob).toBe(zipBlob)
  })

  it('supports pill_id mode without a skill payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(okZipResponse(new Blob(['PK'])))
    vi.stubGlobal('fetch', fetchMock)

    const result = await exportSkill({ pill_id: '550e8400-e29b-41d4-a716-446655440000', format: 'claude' })

    const [, options] = fetchMock.mock.calls[0]
    expect(JSON.parse(options.body)).toEqual({
      pill_id: '550e8400-e29b-41d4-a716-446655440000',
      format: 'claude',
    })
    expect(result.filename).toBe('alchemy-skill-export.zip')
  })

  it('surfaces non-ok responses as ApiError with the stable error code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        errorJsonResponse(503, {
          code: 503,
          error_code: 'skill_export_unavailable',
          message: '导出服务暂不可用',
          request_id: 'request-123',
          data: { stage: 'export', retryable: true },
        })
      )
    )

    const error = await exportSkill({ pill_id: 'x', format: 'codex' }).catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({
      status: 503,
      errorCode: 'skill_export_unavailable',
      message: '导出服务暂不可用',
    })
  })

  it('surfaces network failures as ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network unavailable')))

    const error = await exportSkill({ pill_id: 'x', format: 'codex' }).catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).status).toBe(0)
  })
})
