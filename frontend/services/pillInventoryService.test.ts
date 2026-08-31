import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './api'
import { resolveLegacyPill } from './pillInventoryService'

// 后端信封 { code, message, data }；jsdom 下 request() 把相对路径解析到 http://localhost
function okEnvelope(data: unknown) {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers(),
    json: async () => ({ code: 0, message: 'ok', data }),
  }
}

function notFoundEnvelope() {
  return {
    ok: false,
    status: 404,
    statusText: 'Not Found',
    headers: new Headers(),
    json: async () => ({ code: 404, message: 'pill not found', data: null }),
  }
}

describe('resolveLegacyPill', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const LEGACY_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
  const RECIPE_ID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

  it('queries the explicit legacy entry (GET /pills/:uuid) and returns the recipe pointer', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      okEnvelope({ entity_type: 'recipe', recipe_id: RECIPE_ID })
    )
    vi.stubGlobal('fetch', fetchMock)

    const pointer = await resolveLegacyPill(LEGACY_ID)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    // jsdom 默认 origin 是 http://localhost:3000；断言路径与查询方式即可
    expect(url).toBe(`http://localhost:3000/api/v1/pills/${LEGACY_ID}`)
    expect(options.method).toBe('GET')
    expect(pointer).toEqual({ entity_type: 'recipe', recipe_id: RECIPE_ID })
  })

  it('propagates 404 unchanged (no mapping: caller decides to show not-found)', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(notFoundEnvelope()))

    const error = await resolveLegacyPill(LEGACY_ID).catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).status).toBe(404)
  })
})
