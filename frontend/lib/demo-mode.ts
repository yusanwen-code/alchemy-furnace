/**
 * 演示模式探测(007-demo-mode)
 *
 * SSR 安全:服务端直连 BACKEND_URL,客户端走 /api/v1 代理(开发期 next.config
 * rewrites / 生产期 nginx 反代)。生产环境 output:'export' 下页面在构建期静态
 * 预渲染,此时后端可能未运行,故失败时回退 'real' 不阻塞渲染;实际模式由
 * DemoBanner 客户端组件在浏览器运行期重新探测。
 */

export type AppMode = 'demo' | 'real'

interface HealthData {
  mode?: string
  status?: string
  db?: string
}

/**
 * 探测后端 /api/v1/system/health 的 mode 字段。
 * - 服务端:fetch `${BACKEND_URL}/api/v1/system/health`(默认 http://localhost:8080)
 * - 客户端:fetch `/api/v1/system/health`(相对路径,走代理)
 * 任何异常(HTTP 非 200 / 网络错误 / 解析失败)均回退 'real',绝不抛出。
 */
export async function getMode(): Promise<AppMode> {
  try {
    const isServer = typeof window === 'undefined'
    const base = isServer
      ? (process.env.BACKEND_URL || 'http://localhost:8080')
      : ''
    const res = await fetch(`${base}/api/v1/system/health`, {
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    })
    if (!res.ok) return 'real'
    const body = await res.json()
    const data: HealthData = body?.data ?? body
    return data?.mode === 'demo' ? 'demo' : 'real'
  } catch {
    return 'real'
  }
}
