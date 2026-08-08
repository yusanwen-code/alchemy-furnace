/** @type {import('next').NextConfig} */
const isDev = process.env.NODE_ENV !== 'production'

const nextConfig = {
  // 生产静态导出：产物 out/ 由 nginx 服务（沿用既有部署形态）；
  // 开发环境不导出，动态路由按需渲染（export 会强制校验 generateStaticParams）
  ...(isDev ? {} : { output: 'export' }),
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
