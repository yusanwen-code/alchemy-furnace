import { afterEach, describe, expect, it, vi } from 'vitest'

import { openExternalUrl } from './systemService'

// 桌面端外部链接桥接:POST /api/v1/desktop/open-url(WKWebView 不实现
// target=_blank,点击 GitHub 仓库链接要经 Go 端点交系统浏览器)
function okResponse() {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers(),
    json: async () => ({ code: 0, message: 'ok', data: null }),
  }
}

describe('openExternalUrl', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('posts the external URL to the desktop open-url bridge', async () => {
    const fetchMock = vi.fn().mockResolvedValue(okResponse())
    vi.stubGlobal('fetch', fetchMock)

    await openExternalUrl('https://github.com/yusanwen-code/alchemy-furnace')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    // post() 经 request 的 buildUrl 拼成绝对地址(vitest jsdom origin 即此)
    expect(url).toBe('http://localhost:3000/api/v1/desktop/open-url')
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body)).toEqual({
      url: 'https://github.com/yusanwen-code/alchemy-furnace',
    })
  })

  it('surfaces network failures to the caller', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network unavailable')))
    await expect(openExternalUrl('https://example.com')).rejects.toThrow('network unavailable')
  })
})
