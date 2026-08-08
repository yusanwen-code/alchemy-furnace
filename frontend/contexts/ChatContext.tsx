'use client'

/**
 * 对话状态管理 Context
 * 使用 React Context + useReducer 管理对话相关状态
 * 流式输出通过 WebSocket /api/v1/chat/ws/:session_id（v2 协议）
 * - 每会话一条长连接，支持停止生成（stop/stopped）
 * - 断线自动重连，重连成功后重新拉取消息历史
 * - 断线中断的回复标记「可能不完整」，错误消息以错误气泡内联展示
 */
import React, { createContext, useContext, useReducer, useCallback, useRef, useEffect } from 'react'
import * as chatService from '@/services/chatService'
import type { ConnectionState } from '@/services/chatService'
import type { ChatSession, ChatMessage } from '@/services/types'

/** 对话状态 */
interface ChatState {
  sessions: ChatSession[]
  currentSession: ChatSession | null
  messages: ChatMessage[]
  loading: boolean
  streaming: boolean // 是否正在流式输出
  connectionState: ConnectionState // WebSocket 连接状态
  error: string | null
}

/** 操作类型 */
type ChatAction =
  | { type: 'SET_SESSIONS'; payload: ChatSession[] }
  | { type: 'SET_CURRENT_SESSION'; payload: ChatSession | null }
  | { type: 'SET_MESSAGES'; payload: ChatMessage[] }
  | { type: 'ADD_MESSAGE'; payload: ChatMessage }
  | { type: 'ADD_STREAM_CHUNK'; payload: string } // 追加流式输出内容
  | { type: 'FINISH_STREAM' } // 完成流式输出
  | { type: 'STOP_STREAM' } // 流式输出被停止（保留部分内容）
  | { type: 'MARK_LAST_INCOMPLETE' } // 标记最后一条助手回复「可能不完整」
  | { type: 'ADD_ERROR_MESSAGE'; payload: string } // 内联错误气泡
  | { type: 'ADD_SESSION'; payload: ChatSession }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_STREAMING'; payload: boolean }
  | { type: 'SET_CONNECTION_STATE'; payload: ConnectionState }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'CLEAR_CURRENT' }

/** 初始状态 */
const initialState: ChatState = {
  sessions: [],
  currentSession: null,
  messages: [],
  loading: false,
  streaming: false,
  connectionState: 'disconnected',
  error: null,
}

/** 将流式临时消息转换为正式消息 */
function finalizeStreamMessage(messages: ChatMessage[], patch?: Partial<ChatMessage>): ChatMessage[] {
  const result = [...messages]
  const lastMsg = result[result.length - 1]
  if (lastMsg && lastMsg.role === 'assistant' && lastMsg.id === -1) {
    result[result.length - 1] = { ...lastMsg, id: Date.now(), ...patch }
  }
  return result
}

/** Reducer */
function chatReducer(state: ChatState, action: ChatAction): ChatState {
  switch (action.type) {
    case 'SET_SESSIONS':
      return { ...state, sessions: action.payload, loading: false }
    case 'SET_CURRENT_SESSION':
      return { ...state, currentSession: action.payload, messages: [] }
    case 'SET_MESSAGES':
      return { ...state, messages: action.payload, loading: false }
    case 'ADD_MESSAGE':
      return { ...state, messages: [...state.messages, action.payload] }
    case 'ADD_STREAM_CHUNK': {
      // 追加到最后一条 assistant 消息，如果没有则创建
      const messages = [...state.messages]
      const lastMsg = messages[messages.length - 1]
      if (lastMsg && lastMsg.role === 'assistant' && lastMsg.id === -1) {
        messages[messages.length - 1] = { ...lastMsg, content: lastMsg.content + action.payload }
      } else {
        messages.push({
          id: -1, // 临时 ID
          session_id: state.currentSession?.id || 0,
          role: 'assistant',
          content: action.payload,
          created_at: new Date().toISOString(),
        })
      }
      return { ...state, messages }
    }
    case 'FINISH_STREAM':
      return { ...state, messages: finalizeStreamMessage(state.messages), streaming: false }
    case 'STOP_STREAM':
      // 停止生成：保留部分内容并标记「已停止」
      return { ...state, messages: finalizeStreamMessage(state.messages, { stopped: true }), streaming: false }
    case 'MARK_LAST_INCOMPLETE': {
      const messages = [...state.messages]
      const lastMsg = messages[messages.length - 1]
      if (lastMsg && lastMsg.role === 'assistant' && !lastMsg.is_error) {
        messages[messages.length - 1] = { ...lastMsg, incomplete: true }
      }
      return { ...state, messages }
    }
    case 'ADD_ERROR_MESSAGE': {
      // 服务端错误：以错误气泡内联展示在消息流中
      const errorMessage: ChatMessage = {
        id: Date.now(),
        session_id: state.currentSession?.id || 0,
        role: 'system',
        content: action.payload,
        created_at: new Date().toISOString(),
        is_error: true,
      }
      return { ...state, messages: [...state.messages, errorMessage], streaming: false }
    }
    case 'ADD_SESSION':
      return { ...state, sessions: [action.payload, ...state.sessions], currentSession: action.payload, loading: false }
    case 'SET_LOADING':
      return { ...state, loading: action.payload }
    case 'SET_STREAMING':
      return { ...state, streaming: action.payload }
    case 'SET_CONNECTION_STATE':
      return { ...state, connectionState: action.payload }
    case 'SET_ERROR':
      return { ...state, error: action.payload, loading: false, streaming: false }
    case 'CLEAR_CURRENT':
      return { ...state, currentSession: null, messages: [] }
    default:
      return state
  }
}

/** Context 类型 */
interface ChatContextType {
  state: ChatState
  dispatch: React.Dispatch<ChatAction>
  // 异步操作
  fetchSessions: () => Promise<void>
  createSession: (agentId: number, title?: string) => Promise<ChatSession | null>
  loadMessages: (sessionId: number) => Promise<void>
  /** 建立会话 WebSocket 连接（进入聊天页时调用） */
  connect: (sessionId: number) => void
  /** 主动断开连接（离开聊天页时调用，不触发重连） */
  disconnect: () => void
  streamMessage: (sessionId: number, content: string) => Promise<void>
  /** 停止当前流式生成（发送 stop，等待服务端 stopped 确认） */
  stopStream: () => void
}

const ChatContext = createContext<ChatContextType | null>(null)

/** Provider 组件 */
export function ChatProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(chatReducer, initialState)
  const sessionsRef = useRef<ChatSession[]>([])
  // 断线时是否有未完成的流式内容（用于重连后标记「可能不完整」）
  const interruptedRef = useRef(false)
  // 本轮流式是否已收到内容片段
  const partialReceivedRef = useRef(false)

  // 同步会话列表引用，供 loadMessages 查找当前会话
  useEffect(() => {
    sessionsRef.current = state.sessions
  }, [state.sessions])

  /** 获取会话列表 */
  const fetchSessions = useCallback(async () => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const data = await chatService.listSessions()
      dispatch({ type: 'SET_SESSIONS', payload: data.list || [] })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取会话列表失败' })
    }
  }, [])

  /** 创建会话 */
  const createSession = useCallback(async (agentId: number, title?: string): Promise<ChatSession | null> => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const session = await chatService.createSession({ agent_id: agentId, title })
      dispatch({ type: 'ADD_SESSION', payload: session })
      return session
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '创建会话失败' })
      return null
    }
  }, [])

  /** 加载消息历史并定位当前会话 */
  const loadMessages = useCallback(async (sessionId: number) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      // 定位会话：先查已有列表，查不到则拉取一次会话列表
      let session = sessionsRef.current.find(s => s.id === sessionId)
      if (!session) {
        const data = await chatService.listSessions()
        dispatch({ type: 'SET_SESSIONS', payload: data.list || [] })
        session = (data.list || []).find(s => s.id === sessionId)
      }
      if (session) {
        dispatch({ type: 'SET_CURRENT_SESSION', payload: session })
      }

      const data = await chatService.getMessages(sessionId)
      dispatch({ type: 'SET_MESSAGES', payload: data.list || [] })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '加载消息失败' })
    }
  }, [])

  /** 重连成功后恢复会话：重新拉取消息历史，标记被中断的回复 */
  const reloadAfterReconnect = useCallback(async (sessionId: number) => {
    try {
      const data = await chatService.getMessages(sessionId)
      dispatch({ type: 'SET_MESSAGES', payload: data.list || [] })
      // 断线时若有已生成内容，服务端保存的最后一条回复可能不完整
      if (interruptedRef.current) {
        interruptedRef.current = false
        dispatch({ type: 'MARK_LAST_INCOMPLETE' })
      }
    } catch {
      // 拉取失败：保留现有消息，等待下次重连
    }
  }, [])

  /** 建立会话 WebSocket 连接 */
  const connect = useCallback((sessionId: number) => {
    chatService.connectChat(sessionId, {
      onChunk: (chunk) => {
        partialReceivedRef.current = true
        dispatch({ type: 'ADD_STREAM_CHUNK', payload: chunk })
      },
      onDone: () => {
        partialReceivedRef.current = false
        dispatch({ type: 'FINISH_STREAM' })
      },
      onStopped: () => {
        partialReceivedRef.current = false
        dispatch({ type: 'STOP_STREAM' })
      },
      onError: (error) => {
        partialReceivedRef.current = false
        dispatch({ type: 'ADD_ERROR_MESSAGE', payload: error })
      },
      onConnectionStateChange: (connState) => {
        dispatch({ type: 'SET_CONNECTION_STATE', payload: connState })
      },
      onReconnect: () => {
        void reloadAfterReconnect(sessionId)
      },
      onStreamInterrupted: () => {
        // 流式生成中断线：恢复输入，标记该条回复「可能不完整」
        if (partialReceivedRef.current) {
          interruptedRef.current = true
        }
        partialReceivedRef.current = false
        dispatch({ type: 'FINISH_STREAM' })
        if (interruptedRef.current) {
          dispatch({ type: 'MARK_LAST_INCOMPLETE' })
        }
      },
    })
  }, [reloadAfterReconnect])

  /** 主动断开连接（离开聊天页） */
  const disconnect = useCallback(() => {
    chatService.disconnectChat()
    interruptedRef.current = false
    partialReceivedRef.current = false
    dispatch({ type: 'SET_CONNECTION_STATE', payload: 'disconnected' })
  }, [])

  /** 发送消息（经当前 WebSocket 连接流式接收回复） */
  const streamMessage = useCallback(async (sessionId: number, content: string) => {
    // 先添加用户消息
    const userMessage: ChatMessage = {
      id: Date.now(),
      session_id: sessionId,
      role: 'user',
      content,
      created_at: new Date().toISOString(),
    }
    dispatch({ type: 'ADD_MESSAGE', payload: userMessage })

    const sent = chatService.sendChatMessage(content)
    if (!sent) {
      dispatch({ type: 'ADD_ERROR_MESSAGE', payload: '连接未就绪，请等待重连成功后再发送' })
      return
    }
    partialReceivedRef.current = false
    dispatch({ type: 'SET_STREAMING', payload: true })
  }, [])

  /** 停止当前流式生成（等待服务端 stopped 确认后恢复输入） */
  const stopStream = useCallback(() => {
    chatService.sendStop()
  }, [])

  return (
    <ChatContext.Provider
      value={{
        state,
        dispatch,
        fetchSessions,
        createSession,
        loadMessages,
        connect,
        disconnect,
        streamMessage,
        stopStream,
      }}
    >
      {children}
    </ChatContext.Provider>
  )
}

/** Hook */
export function useChat(): ChatContextType {
  const context = useContext(ChatContext)
  if (!context) {
    throw new Error('useChat must be used within a ChatProvider')
  }
  return context
}
