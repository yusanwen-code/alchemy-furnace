import ChatSessionPage from './session-page'

// 静态导出占位：真实 sessionId 由 nginx SPA fallback + 客户端 useParams 提供
export function generateStaticParams() {
  return [{ sessionId: '_' }]
}

export default function Page() {
  return <ChatSessionPage />
}
