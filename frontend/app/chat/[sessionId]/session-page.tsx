'use client'

/**
 * 论道 - 已有会话（带 sessionId）
 */
import { useParams } from 'next/navigation'
import { ChatView } from '../chat-view'

export default function ChatSessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  return <ChatView sessionId={sessionId} />
}
