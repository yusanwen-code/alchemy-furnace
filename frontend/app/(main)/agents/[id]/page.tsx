import AgentDetailPage from './agent-detail'

// 静态导出占位：真实 id 由 nginx SPA fallback + 客户端 useParams 提供
export function generateStaticParams() {
  return [{ id: '_' }]
}

export default function Page() {
  return <AgentDetailPage />
}
