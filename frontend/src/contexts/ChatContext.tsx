/**
 * 对话状态管理 Context
 * 使用 React Context + useReducer 管理对话相关状态
 * 包含会话列表、当前会话、消息列表、流式输出状态等
 */
import React, { createContext, useContext, useReducer, useCallback, useRef } from 'react'
import * as chatService from '@/services/chatService'
import type { ChatSession, ChatMessage, Source } from '@/services/types'

/** 对话状态 */
interface ChatState {
  sessions: ChatSession[]
  currentSession: ChatSession | null
  messages: ChatMessage[]
  loading: boolean
  streaming: boolean // 是否正在流式输出
  error: string | null
}

/** 操作类型 */
type ChatAction =
  | { type: 'SET_SESSIONS'; payload: ChatSession[] }
  | { type: 'SET_CURRENT_SESSION'; payload: ChatSession | null }
  | { type: 'SET_MESSAGES'; payload: ChatMessage[] }
  | { type: 'ADD_MESSAGE'; payload: ChatMessage }
  | { type: 'ADD_STREAM_CHUNK'; payload: string } // 追加流式输出内容
  | { type: 'FINISH_STREAM'; payload: { content: string; sources?: Source[] } } // 完成流式输出
  | { type: 'ADD_SESSION'; payload: ChatSession }
  | { type: 'REMOVE_SESSION'; payload: number }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_STREAMING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'CLEAR_CURRENT' }

/** 初始状态 */
const initialState: ChatState = {
  sessions: [],
  currentSession: null,
  messages: [],
  loading: false,
  streaming: false,
  error: null,
}

/** Reducer */
function chatReducer(state: ChatState, action: ChatAction): ChatState {
  switch (action.type) {
    case 'SET_SESSIONS':
      return { ...state, sessions: action.payload, loading: false }
    case 'SET_CURRENT_SESSION':
      return { ...state, currentSession: action.payload, messages: [] }
    case 'SET_MESSAGES':
      return { ...state, messages: action.payload }
    case 'ADD_MESSAGE':
      return { ...state, messages: [...state.messages, action.payload] }
    case 'ADD_STREAM_CHUNK': {
      // 追加到最后一条 assistant 消息，如果没有则创建
      const messages = [...state.messages]
      const lastMsg = messages[messages.length - 1]
      if (lastMsg && lastMsg.role === 'assistant' && lastMsg.id === -1) {
        lastMsg.content += action.payload
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
    case 'FINISH_STREAM': {
      // 将流式消息转换为正式消息
      const messages = [...state.messages]
      const lastMsg = messages[messages.length - 1]
      if (lastMsg && lastMsg.role === 'assistant' && lastMsg.id === -1) {
        lastMsg.id = Date.now() // 分配正式 ID
        lastMsg.content = action.payload.content
        lastMsg.sources = action.payload.sources
      }
      return { ...state, messages, streaming: false }
    }
    case 'ADD_SESSION':
      return { ...state, sessions: [action.payload, ...state.sessions], currentSession: action.payload }
    case 'REMOVE_SESSION':
      return {
        ...state,
        sessions: state.sessions.filter(s => s.id !== action.payload),
        currentSession: state.currentSession?.id === action.payload ? null : state.currentSession,
      }
    case 'SET_LOADING':
      return { ...state, loading: action.payload }
    case 'SET_STREAMING':
      return { ...state, streaming: action.payload }
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
  createSession: (agentId: number, title?: string) => Promise<void>
  deleteSession: (id: number) => Promise<void>
  loadMessages: (sessionId: number) => Promise<void>
  sendMessage: (sessionId: number, content: string) => Promise<void>
  streamMessage: (sessionId: number, content: string) => Promise<void>
  cancelStream: () => void
}

const ChatContext = createContext<ChatContextType | null>(null)

/** Provider 组件 */
export function ChatProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(chatReducer, initialState)
  const abortRef = useRef(false)

  /** 获取会话列表 */
  const fetchSessions = useCallback(async () => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const sessions = await chatService.getSessions()
      dispatch({ type: 'SET_SESSIONS', payload: sessions })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取会话列表失败' })
    }
  }, [])

  /** 创建会话 */
  const createSession = useCallback(async (agentId: number, title?: string) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const session = await chatService.createSession({ agent_id: agentId, title })
      dispatch({ type: 'ADD_SESSION', payload: session })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '创建会话失败' })
    }
  }, [])

  /** 删除会话 */
  const deleteSession = useCallback(async (id: number) => {
    try {
      await chatService.deleteSession(id)
      dispatch({ type: 'REMOVE_SESSION', payload: id })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '删除会话失败' })
    }
  }, [])

  /** 加载消息历史 */
  const loadMessages = useCallback(async (sessionId: number) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const messages = await chatService.getMessages(sessionId)
      dispatch({ type: 'SET_MESSAGES', payload: messages })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '加载消息失败' })
    }
  }, [])

  /** 发送消息（非流式） */
  const sendMessage = useCallback(async (sessionId: number, content: string) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const { userMessage, assistantMessage } = await chatService.sendMessage(sessionId, content)
      dispatch({ type: 'ADD_MESSAGE', payload: userMessage })
      dispatch({ type: 'ADD_MESSAGE', payload: assistantMessage })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '发送消息失败' })
    }
  }, [])

  /** 流式发送消息 */
  const streamMessage = useCallback(async (sessionId: number, content: string) => {
    abortRef.current = false
    dispatch({ type: 'SET_STREAMING', payload: true })

    // 先添加用户消息
    const userMessage: ChatMessage = {
      id: Date.now(),
      session_id: sessionId,
      role: 'user',
      content,
      created_at: new Date().toISOString(),
    }
    dispatch({ type: 'ADD_MESSAGE', payload: userMessage })

    try {
      const generator = chatService.streamMessage(sessionId, content)
      let finalContent = ''
      let finalSources: Source[] | undefined

      for await (const chunk of generator) {
        if (abortRef.current) {
          dispatch({ type: 'SET_STREAMING', payload: false })
          return
        }

        if (chunk.type === 'chunk' && chunk.content) {
          dispatch({ type: 'ADD_STREAM_CHUNK', payload: chunk.content })
          finalContent += chunk.content
        } else if (chunk.type === 'done') {
          finalContent = chunk.content || finalContent
          finalSources = chunk.sources
        } else if (chunk.type === 'error') {
          dispatch({ type: 'SET_ERROR', payload: chunk.error || '流式输出出错' })
          return
        }
      }

      dispatch({ type: 'FINISH_STREAM', payload: { content: finalContent, sources: finalSources } })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '流式输出失败' })
    }
  }, [])

  /** 取消流式输出 */
  const cancelStream = useCallback(() => {
    abortRef.current = true
    dispatch({ type: 'SET_STREAMING', payload: false })
  }, [])

  return (
    <ChatContext.Provider
      value={{
        state,
        dispatch,
        fetchSessions,
        createSession,
        deleteSession,
        loadMessages,
        sendMessage,
        streamMessage,
        cancelStream,
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
