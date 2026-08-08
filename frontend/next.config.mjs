/** @type {import('next').NextConfig} */
const isDev = process.env.NODE_ENV !== 'production'

const nextConfig = {
  // 静态导出：产物 out/ 由 nginx 服务（沿用既有部署形态）
  output: 'export',
  images: { unoptimized: true },
  // 开发环境将 REST API 代理到 Go 后端；WebSocket 走 NEXT_PUBLIC_WS_BASE_URL 直连
  ...(isDev
    ? {
        rewrites: async () => [
          {
            source: '/api/:path*',
            destination: 'http://localhost:8080/api/:path*',
          },
        ],
      }
    : {}),
}

export default nextConfig
