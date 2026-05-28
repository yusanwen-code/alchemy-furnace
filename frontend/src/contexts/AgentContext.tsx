/**
 * 道人状态管理 Context
 * 使用 React Context + useReducer 管理 AI Agent 相关状态
 * 包含道人列表、当前选中的道人、已服用金丹等
 */
import React, { createContext, useContext, useReducer, useCallback } from 'react'
import * as agentService from '@/services/agentService'
import type { Agent, Pill } from '@/services/types'

/** 道人状态 */
interface AgentState {
  agents: Agent[]
  currentAgent: Agent | null
  currentAgentPills: Pill[]
  loading: boolean
  error: string | null
}

/** 操作类型 */
type AgentAction =
  | { type: 'SET_AGENTS'; payload: Agent[] }
  | { type: 'SET_CURRENT_AGENT'; payload: Agent | null }
  | { type: 'SET_CURRENT_AGENT_PILLS'; payload: Pill[] }
  | { type: 'ADD_AGENT'; payload: Agent }
  | { type: 'UPDATE_AGENT'; payload: Agent }
  | { type: 'REMOVE_AGENT'; payload: number }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }

/** 初始状态 */
const initialState: AgentState = {
  agents: [],
  currentAgent: null,
  currentAgentPills: [],
  loading: false,
  error: null,
}

/** Reducer */
function agentReducer(state: AgentState, action: AgentAction): AgentState {
  switch (action.type) {
    case 'SET_AGENTS':
      return { ...state, agents: action.payload, loading: false }
    case 'SET_CURRENT_AGENT':
      return { ...state, currentAgent: action.payload }
    case 'SET_CURRENT_AGENT_PILLS':
      return { ...state, currentAgentPills: action.payload }
    case 'ADD_AGENT':
      return { ...state, agents: [action.payload, ...state.agents] }
    case 'UPDATE_AGENT':
      return {
        ...state,
        agents: state.agents.map(a => (a.id === action.payload.id ? action.payload : a)),
        currentAgent: state.currentAgent?.id === action.payload.id ? action.payload : state.currentAgent,
      }
    case 'REMOVE_AGENT':
      return {
        ...state,
        agents: state.agents.filter(a => a.id !== action.payload),
        currentAgent: state.currentAgent?.id === action.payload ? null : state.currentAgent,
      }
    case 'SET_LOADING':
      return { ...state, loading: action.payload }
    case 'SET_ERROR':
      return { ...state, error: action.payload, loading: false }
    default:
      return state
  }
}

/** Context 类型 */
interface AgentContextType {
  state: AgentState
  dispatch: React.Dispatch<AgentAction>
  // 异步操作
  fetchAgents: () => Promise<void>
  fetchAgent: (id: number) => Promise<void>
  fetchAgentPills: (agentId: number) => Promise<void>
  addAgent: (name: string, modelName: string, personality?: string) => Promise<void>
  removeAgent: (id: number) => Promise<void>
  bindPill: (agentId: number, pillId: number) => Promise<void>
  unbindPill: (agentId: number, pillId: number) => Promise<void>
}

const AgentContext = createContext<AgentContextType | null>(null)

/** Provider 组件 */
export function AgentProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(agentReducer, initialState)

  /** 获取道人列表 */
  const fetchAgents = useCallback(async () => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const agents = await agentService.getAgents()
      dispatch({ type: 'SET_AGENTS', payload: agents })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取道人列表失败' })
    }
  }, [])

  /** 获取单个道人 */
  const fetchAgent = useCallback(async (id: number) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const agent = await agentService.getAgent(id)
      dispatch({ type: 'SET_CURRENT_AGENT', payload: agent })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取道人详情失败' })
    }
  }, [])

  /** 获取道人已服用金丹 */
  const fetchAgentPills = useCallback(async (agentId: number) => {
    try {
      const pills = await agentService.getAgentPills(agentId)
      dispatch({ type: 'SET_CURRENT_AGENT_PILLS', payload: pills })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取金丹列表失败' })
    }
  }, [])

  /** 创建道人 */
  const addAgent = useCallback(async (name: string, modelName: string, personality?: string) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const agent = await agentService.createAgent({ name, model_name: modelName, personality })
      dispatch({ type: 'ADD_AGENT', payload: agent })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '创建道人失败' })
    }
  }, [])

  /** 删除道人 */
  const removeAgent = useCallback(async (id: number) => {
    try {
      await agentService.deleteAgent(id)
      dispatch({ type: 'REMOVE_AGENT', payload: id })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '删除道人失败' })
    }
  }, [])

  /** 服用金丹（绑定） */
  const bindPill = useCallback(async (agentId: number, pillId: number) => {
    try {
      await agentService.bindPill(agentId, pillId)
      // 刷新已服用金丹列表
      const pills = await agentService.getAgentPills(agentId)
      dispatch({ type: 'SET_CURRENT_AGENT_PILLS', payload: pills })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '服用金丹失败' })
    }
  }, [])

  /** 解除金丹绑定 */
  const unbindPill = useCallback(async (agentId: number, pillId: number) => {
    try {
      await agentService.unbindPill(agentId, pillId)
      // 刷新已服用金丹列表
      const pills = await agentService.getAgentPills(agentId)
      dispatch({ type: 'SET_CURRENT_AGENT_PILLS', payload: pills })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '解除绑定失败' })
    }
  }, [])

  return (
    <AgentContext.Provider
      value={{
        state,
        dispatch,
        fetchAgents,
        fetchAgent,
        fetchAgentPills,
        addAgent,
        removeAgent,
        bindPill,
        unbindPill,
      }}
    >
      {children}
    </AgentContext.Provider>
  )
}

/** Hook */
export function useAgent(): AgentContextType {
  const context = useContext(AgentContext)
  if (!context) {
    throw new Error('useAgent must be used within a AgentProvider')
  }
  return context
}
