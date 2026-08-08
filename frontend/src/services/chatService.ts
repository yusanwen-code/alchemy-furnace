/**
 * 对话服务 - 会话管理、消息 API 与 WebSocket 流式对话（v2 协议）
 * 对接后端 /api/v1/chat/sessions 与 WebSocket /api/v1/chat/ws/:session_id
 *
 * WebSocket v2 协议要点：
 * - 每个会话维持一条长连接，消息（{ content }）经同一连接发送
 * - 停止生成：发送 { type: "stop" }，服务端以 { type: "stopped" } 确认（不关闭连接）
 * - 错误消息 { type: "error", content } 为可读中文描述，需恢复输入状态
 * - 断线重连：指数退避 1s/2s/4s/8s/16s，最多 5 次；页面隐藏时暂停重连
 * - 用户主动离开聊天页为正常关闭，不触发重连
 * - 重连成功后需重新拉取会话历史（消息由服务端持久化）
 * - 心跳为 WS 协议层 ping/pong，浏览器自动应答，无需应用层处理
 */
import { get, post, buildWsUrl } from './api'
import type { ChatSession, ChatMessage, CreateSessionRequest, PagedList, ListParams, WSMessage } from './types'

/**
 * 获取会话列表
 */
export function listSessions(params: ListParams = {}): Promise<PagedList<ChatSession>> {
  return get<PagedList<ChatSession>>('/chat/sessions', {
    page: params.page ?? 1,
    page_size: params.page_size ?? 100,
  })
}

/**
 * 创建会话
 */
export function createSession(data: CreateSessionRequest): Promise<ChatSession> {
  return post<ChatSession>('/chat/sessions', data)
}

/**
 * 获取会话消息历史（按时间正序）
 */
export function getMessages(sessionId: number, params: ListParams = {}): Promise<PagedList<ChatMessage>> {
  return get<PagedList<ChatMessage>>(`/chat/sessions/${sessionId}/messages`, {
    page: params.page ?? 1,
    page_size: params.page_size ?? 200,
  })
}

/** WebSocket 连接状态 */
export type ConnectionState = 'connected' | 'connecting' | 'disconnected' | 'failed'

/** 流式对话连接回调 */
export interface ChatConnectionHandlers {
  /** 收到内容片段 */
  onChunk: (content: string) => void
  /** 流式输出完成 */
  onDone: () => void
  /** 生成已被停止（此前内容已保存） */
  onStopped: () => void
  /** 服务端错误（可读中文描述） */
  onError: (error: string) => void
  /** 连接状态变化 */
  onConnectionStateChange: (state: ConnectionState) => void
  /** 重连成功（应重新拉取会话历史） */
  onReconnect: () => void
  /** 流式生成中断线（该条回复可能不完整） */
  onStreamInterrupted: () => void
}

/** 重连退避间隔（最多 5 次） */
const BACKOFF_MS = [1000, 2000, 4000, 8000, 16000]
const MAX_ATTEMPTS = 5

/** 单会话 WebSocket 连接 */
interface Connection {
  sessionId: number
  ws: WebSocket | null
  handlers: ChatConnectionHandlers
  /** 已尝试的重连次数 */
  attempts: number
  reconnectTimer: number | null
  /** 用户主动关闭（离开聊天页），不触发重连 */
  userClosed: boolean
  /** 是否有流式生成进行中 */
  streamInProgress: boolean
  state: ConnectionState
  /** 页面隐藏导致重连暂停 */
  paused: boolean
}

// 当前活跃连接（同一时间只允许一条）
let conn: Connection | null = null

/** 更新连接状态并通知 */
function setState(c: Connection, state: ConnectionState): void {
  if (c.state === state) return
  c.state = state
  c.handlers.onConnectionStateChange(state)
}

/** 清除重连定时器 */
function clearReconnectTimer(c: Connection): void {
  if (c.reconnectTimer !== null) {
    window.clearTimeout(c.reconnectTimer)
    c.reconnectTimer = null
  }
}

/**
 * 建立（或复用）指定会话的 WebSocket 连接
 * 相同会话重复调用仅更新回调；不同会话则切换连接
 */
export function connectChat(sessionId: number, handlers: ChatConnectionHandlers): void {
  // 相同会话且连接仍存在：仅替换回调
  if (conn && conn.sessionId === sessionId && !conn.userClosed) {
    conn.handlers = handlers
    handlers.onConnectionStateChange(conn.state)
    return
  }

  disconnectChat()

  const c: Connection = {
    sessionId,
    ws: null,
    handlers,
    attempts: 0,
    reconnectTimer: null,
    userClosed: false,
    streamInProgress: false,
    state: 'connecting',
    paused: false,
  }
  conn = c
  setState(c, 'connecting')
  openSocket(c)
}

/** 打开 WebSocket 并绑定事件 */
function openSocket(c: Connection): void {
  const ws = new WebSocket(buildWsUrl(`/chat/ws/${c.sessionId}`))
  c.ws = ws

  ws.onopen = () => {
    if (conn !== c) return
    const isReconnect = c.attempts > 0
    c.attempts = 0
    setState(c, 'connected')
    if (isReconnect) {
      // 重连成功：会话状态存于服务端，通知调用方重新拉取历史
      c.handlers.onReconnect()
    }
  }

  ws.onmessage = (event) => {
    if (conn !== c) return
    try {
      const msg = JSON.parse(event.data as string) as WSMessage
      if (msg.type === 'chunk' && msg.content) {
        c.handlers.onChunk(msg.content)
      } else if (msg.type === 'done') {
        c.streamInProgress = false
        c.handlers.onDone()
      } else if (msg.type === 'stopped') {
        c.streamInProgress = false
        c.handlers.onStopped()
      } else if (msg.type === 'error') {
        c.streamInProgress = false
        c.handlers.onError(msg.content || '论道出错了')
      }
    } catch {
      // 忽略无法解析的消息
    }
  }

  ws.onerror = () => {
    // 浏览器随后会触发 onclose，统一在 onclose 中处理重连
  }

  ws.onclose = () => {
    if (conn !== c) return
    c.ws = null
    if (c.userClosed) {
      setState(c, 'disconnected')
      return
    }
    // 意外断线：若流式生成进行中，该条回复可能不完整
    if (c.streamInProgress) {
      c.streamInProgress = false
      c.handlers.onStreamInterrupted()
    }
    scheduleReconnect(c)
  }
}

/** 按指数退避调度重连；页面隐藏时暂停，恢复可见后立即重连 */
function scheduleReconnect(c: Connection): void {
  if (conn !== c || c.userClosed) return
  if (c.attempts >= MAX_ATTEMPTS) {
    setState(c, 'failed')
    return
  }
  setState(c, 'connecting')
  const delay = BACKOFF_MS[Math.min(c.attempts, BACKOFF_MS.length - 1)]
  c.attempts += 1

  if (document.hidden) {
    c.paused = true
    return
  }

  c.reconnectTimer = window.setTimeout(() => {
    c.reconnectTimer = null
    if (conn === c && !c.userClosed) {
      openSocket(c)
    }
  }, delay)
}

// 页面恢复可见时，立即恢复被暂停的重连
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    const c = conn
    if (!document.hidden && c && c.paused && !c.userClosed && c.state !== 'connected') {
      c.paused = false
      clearReconnectTimer(c)
      openSocket(c)
    }
  })
}

/**
 * 通过当前连接发送用户消息
 * @returns 是否发送成功（连接未就绪时返回 false）
 */
export function sendChatMessage(content: string): boolean {
  if (!conn || !conn.ws || conn.ws.readyState !== WebSocket.OPEN || conn.userClosed) {
    return false
  }
  conn.ws.send(JSON.stringify({ content }))
  conn.streamInProgress = true
  return true
}

/**
 * 停止当前流式生成（发送 { type: "stop" }，不关闭连接）
 * 仅在生成期间有效，服务端将以 { type: "stopped" } 确认
 */
export function sendStop(): void {
  if (!conn || !conn.ws || conn.ws.readyState !== WebSocket.OPEN) return
  conn.ws.send(JSON.stringify({ type: 'stop' }))
}

/**
 * 主动断开连接（离开聊天页），不触发重连
 */
export function disconnectChat(): void {
  if (!conn) return
  const c = conn
  conn = null
  c.userClosed = true
  c.streamInProgress = false
  clearReconnectTimer(c)
  if (c.ws) {
    try {
      c.ws.close()
    } catch {
      // 忽略关闭异常
    }
    c.ws = null
  }
}
