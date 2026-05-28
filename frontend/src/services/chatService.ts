/**
 * 对话服务 - 会话管理和消息 API
 * 提供会话的增删改查及 WebSocket 流式对话（演示模式使用 Mock 数据）
 */
import { mockDelay } from './api'
import { mockSessions, mockMessages, mockAgents } from './mockData'
import type { ChatSession, ChatMessage, CreateSessionRequest, WSMessage } from './types'

let sessions = [...mockSessions]
let messages = { ...mockMessages }
let nextSessionId = Math.max(...sessions.map(s => s.id)) + 1
let nextMessageId = Math.max(...Object.values(messages).flat().map(m => m.id)) + 1

/**
 * 获取会话列表
 */
export async function getSessions(): Promise<ChatSession[]> {
  await mockDelay()
  return [...sessions].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
}

/**
 * 创建会话
 */
export async function createSession(data: CreateSessionRequest): Promise<ChatSession> {
  await mockDelay()
  const agent = mockAgents.find(a => a.id === data.agent_id)
  const session: ChatSession = {
    id: nextSessionId++,
    agent_id: data.agent_id,
    title: data.title || `与${agent?.name || '未知道人'}的对话`,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
  sessions.push(session)
  messages[session.id] = []
  return { ...session }
}

/**
 * 删除会话
 */
export async function deleteSession(id: number): Promise<void> {
  await mockDelay()
  sessions = sessions.filter(s => s.id !== id)
  delete messages[id]
}

/**
 * 获取会话消息历史
 */
export async function getMessages(sessionId: number): Promise<ChatMessage[]> {
  await mockDelay()
  return [...(messages[sessionId] || [])]
}

/**
 * 发送消息（演示模式：模拟 AI 回复）
 */
export async function sendMessage(
  sessionId: number,
  content: string
): Promise<{ userMessage: ChatMessage; assistantMessage: ChatMessage }> {
  await mockDelay(500)

  // 创建用户消息
  const userMessage: ChatMessage = {
    id: nextMessageId++,
    session_id: sessionId,
    role: 'user',
    content,
    created_at: new Date().toISOString(),
  }

  // 创建 AI 回复（模拟）
  const assistantMessage: ChatMessage = {
    id: nextMessageId++,
    session_id: sessionId,
    role: 'assistant',
    content: generateMockReply(content),
    sources: [
      { content: '相关引用内容示例...', score: 0.92, metadata: { filename: '示例文档.docx', page: 1 } },
    ],
    created_at: new Date().toISOString(),
  }

  if (!messages[sessionId]) messages[sessionId] = []
  messages[sessionId].push(userMessage, assistantMessage)

  // 更新会话时间
  const sessionIndex = sessions.findIndex(s => s.id === sessionId)
  if (sessionIndex !== -1) {
    sessions[sessionIndex].updated_at = new Date().toISOString()
  }

  return { userMessage, assistantMessage }
}

/**
 * 模拟流式对话（WebSocket 替代方案）
 * 返回 AsyncGenerator，逐字输出 AI 回复
 */
export async function* streamMessage(sessionId: number, content: string): AsyncGenerator<WSMessage> {
  // 先保存用户消息
  const userMessage: ChatMessage = {
    id: nextMessageId++,
    session_id: sessionId,
    role: 'user',
    content,
    created_at: new Date().toISOString(),
  }
  if (!messages[sessionId]) messages[sessionId] = []
  messages[sessionId].push(userMessage)

  // 模拟 AI 思考延迟
  await new Promise(resolve => setTimeout(resolve, 800))

  // 生成回复内容
  const reply = generateMockReply(content)
  const chunkSize = 3 // 每次输出 3 个字符

  // 逐字输出
  for (let i = 0; i < reply.length; i += chunkSize) {
    const chunk = reply.slice(i, i + chunkSize)
    await new Promise(resolve => setTimeout(resolve, 40 + Math.random() * 30))
    yield {
      type: 'chunk',
      content: chunk,
    }
  }

  // 完成信号
  const sources = [
    { content: '相关引用内容示例...', score: 0.92, metadata: { filename: '示例文档.docx', page: 1 } },
  ]
  yield {
    type: 'done',
    content: reply,
    sources,
  }

  // 保存 AI 消息
  const assistantMessage: ChatMessage = {
    id: nextMessageId++,
    session_id: sessionId,
    role: 'assistant',
    content: reply,
    sources,
    created_at: new Date().toISOString(),
  }
  messages[sessionId].push(assistantMessage)

  // 更新会话时间
  const sessionIndex = sessions.findIndex(s => s.id === sessionId)
  if (sessionIndex !== -1) {
    sessions[sessionIndex].updated_at = new Date().toISOString()
  }
}

/**
 * 根据用户输入生成模拟回复
 */
function generateMockReply(userContent: string): string {
  const replies = [
    `道友问得好！关于"${userContent.slice(0, 20)}"，老夫认为这其中大有玄机。\n\n首先，需明白**道生一，一生二，二生三，三生万物**的道理。修行之路切忌急功近利，当循序渐进。\n\n具体而言，建议从以下几个方面入手：\n\n1. **静心** - 排除杂念，方能感悟天地灵气\n2. **筑基** - 打好根基，后续修炼方能事半功倍\n3. **实践** - 理论结合实践，在实战中检验所学\n\n道友若有更多疑问，随时来询。`,

    `哈哈，此问正合我意！让老夫为你推演一番。\n\n> **天机示警**：修行之道，贵在持之以恒。\n\n关于你的问题，我的见解如下：\n\n- 若以\`python\`修炼为例，切记**模块化**之道，不可将所有功法写在一处\n- 心法修炼如同写**单元测试**，需反复验证，确保根基稳固\n- 遇到瓶颈时，不妨参考前人经验，正所谓"他山之石，可以攻玉"\n\n\`\`\`python\ndef cultivate():\n    while True:\n        meditate()\n        practice()\n        if breakthrough():\n            celebrate()\n\`\`\`\n\n道友以为如何？`,

    `善哉！道友有此觉悟，修行之路可期。\n\n老夫查阅了金丹阁中的典籍，发现以下几处与你所问相关：\n\n## 相关记载\n\n**《修仙功法总纲》** 第12页记载：\n> "筑基者，修仙之本也。需打通奇经八脉，凝聚丹田之气。"\n\n**《心法口诀集》** 第28页亦有提及：\n> "心魔劫者，修行者必经之路。当以清心诀化解之。"\n\n## 我的建议\n\n道友可根据自身情况，选择适合的修炼路径。切记：**道法自然，不可强求**。`,
  ]

  // 根据用户内容长度选择一个回复
  const index = userContent.length % replies.length
  return replies[index]
}
