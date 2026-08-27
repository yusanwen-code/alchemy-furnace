'use client'

/**
 * 金丹状态管理 Context
 * 使用 React Context + useReducer 管理金丹（语言模式技能包）相关状态
 */
import React, { createContext, useContext, useReducer, useCallback, useRef } from 'react'
import * as pillService from '@/services/pillService'
import { ApiError } from '@/services/api'
import type { Pill, CreatePillRequest, UpdatePillRequest, PillListParams } from '@/services/types'

/** 金丹详情加载状态(按 ID 归属;只有 API 明确 404 才判定不存在) */
type DetailStatus = 'idle' | 'loading' | 'ready' | 'not-found' | 'error'

interface DetailLoadState {
  id: string | null
  status: DetailStatus
  error: string | null
}

/** 金丹状态 */
interface PillState {
  pills: Pill[]
  total: number
  currentPill: Pill | null
  loading: boolean
  error: string | null
  detailLoad: DetailLoadState
}

/** 操作类型 */
type PillAction =
  | { type: 'SET_PILLS'; payload: { list: Pill[]; total: number } }
  | { type: 'SET_CURRENT_PILL'; payload: Pill | null }
  | { type: 'ADD_PILL'; payload: Pill }
  | { type: 'UPDATE_PILL'; payload: Pill }
  | { type: 'REMOVE_PILL'; payload: string }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'DETAIL_START'; payload: { id: string } }
  | { type: 'DETAIL_READY'; payload: { id: string; pill: Pill } }
  | { type: 'DETAIL_NOT_FOUND'; payload: { id: string } }
  | { type: 'DETAIL_ERROR'; payload: { id: string; error: string } }

/** 初始状态 */
const initialState: PillState = {
  pills: [],
  total: 0,
  currentPill: null,
  loading: false,
  error: null,
  detailLoad: { id: null, status: 'idle', error: null },
}

/** 详情完成类 action 的竞态守卫:只在 id 归属匹配时接受结果,防旧请求覆盖新页 */
function acceptDetail(state: PillState, id: string): boolean {
  return state.detailLoad.id === id
}

/** Reducer */
function pillReducer(state: PillState, action: PillAction): PillState {
  switch (action.type) {
    case 'SET_PILLS':
      return { ...state, pills: action.payload.list, total: action.payload.total, loading: false }
    case 'SET_CURRENT_PILL':
      return { ...state, currentPill: action.payload, loading: false }
    case 'ADD_PILL':
      return { ...state, pills: [action.payload, ...state.pills], total: state.total + 1 }
    case 'UPDATE_PILL':
      return {
        ...state,
        pills: state.pills.map(p => (p.id === action.payload.id ? action.payload : p)),
        currentPill: state.currentPill?.id === action.payload.id ? action.payload : state.currentPill,
      }
    case 'REMOVE_PILL':
      return {
        ...state,
        pills: state.pills.filter(p => p.id !== action.payload),
        currentPill: state.currentPill?.id === action.payload ? null : state.currentPill,
      }
    case 'SET_LOADING':
      return { ...state, loading: action.payload }
    case 'SET_ERROR':
      return { ...state, error: action.payload, loading: false }
    case 'DETAIL_START':
      return { ...state, detailLoad: { id: action.payload.id, status: 'loading', error: null } }
    case 'DETAIL_READY':
      if (!acceptDetail(state, action.payload.id)) return state
      // 守卫通过才原子落库:状态与金丹一起更新,防旧请求的 success 污染新页
      return {
        ...state,
        detailLoad: { id: action.payload.id, status: 'ready', error: null },
        currentPill: action.payload.pill,
      }
    case 'DETAIL_NOT_FOUND':
      if (!acceptDetail(state, action.payload.id)) return state
      return { ...state, detailLoad: { id: action.payload.id, status: 'not-found', error: null } }
    case 'DETAIL_ERROR':
      if (!acceptDetail(state, action.payload.id)) return state
      return { ...state, detailLoad: { id: action.payload.id, status: 'error', error: action.payload.error } }
    default:
      return state
  }
}

/** Context 类型 */
interface PillContextType {
  state: PillState
  dispatch: React.Dispatch<PillAction>
  // 异步操作
  fetchPills: (params?: PillListParams) => Promise<void>
  fetchPill: (id: string) => Promise<void>
  /** 创建金丹；失败 dispatch SET_ERROR 并原样抛出错误（ApiError），不得以 null 静默 */
  addPill: (data: CreatePillRequest) => Promise<Pill>
  /** 更新金丹；失败 dispatch SET_ERROR 并原样抛出错误（ApiError），不得以 null 静默 */
  editPill: (id: string, data: UpdatePillRequest) => Promise<Pill>
  /** 删除金丹；失败 dispatch SET_ERROR 并原样抛出错误（ApiError），不得以 false 静默 */
  removePill: (id: string) => Promise<void>
}

const PillContext = createContext<PillContextType | null>(null)

/** Provider 组件 */
export function PillProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(pillReducer, initialState)
  const listRequestRef = useRef(0)

  /** 获取金丹列表 */
  const fetchPills = useCallback(async (params: PillListParams = {}) => {
    const requestId = ++listRequestRef.current
    dispatch({ type: 'SET_ERROR', payload: null })
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const data = await pillService.listPills(params)
      if (requestId !== listRequestRef.current) return
      dispatch({ type: 'SET_PILLS', payload: { list: data.list || [], total: data.total } })
    } catch (error) {
      if (requestId !== listRequestRef.current) return
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取金丹列表失败' })
    }
  }, [])

  /** 获取单个金丹(按 ID 归属;只有 API 明确 404 才判定"不存在或已删除";旧请求晚到不覆盖新页) */
  const fetchPill = useCallback(async (id: string) => {
    dispatch({ type: 'DETAIL_START', payload: { id } })
    try {
      const pill = await pillService.getPill(id)
      dispatch({ type: 'DETAIL_READY', payload: { id, pill } })
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        dispatch({ type: 'DETAIL_NOT_FOUND', payload: { id } })
      } else {
        dispatch({
          type: 'DETAIL_ERROR',
          payload: { id, error: error instanceof Error ? error.message : '获取金丹详情失败' },
        })
      }
    }
  }, [])

  /** 创建金丹（失败抛错，调用方必须自行捕获） */
  const addPill = useCallback(async (data: CreatePillRequest): Promise<Pill> => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const pill = await pillService.createPill(data)
      dispatch({ type: 'ADD_PILL', payload: pill })
      dispatch({ type: 'SET_LOADING', payload: false })
      return pill
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '创建金丹失败' })
      throw error
    }
  }, [])

  /** 更新金丹（失败抛错，调用方必须自行捕获） */
  const editPill = useCallback(async (id: string, data: UpdatePillRequest): Promise<Pill> => {
    try {
      const pill = await pillService.updatePill(id, data)
      dispatch({ type: 'UPDATE_PILL', payload: pill })
      return pill
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '更新金丹失败' })
      throw error
    }
  }, [])

  /** 删除金丹（失败抛错，调用方必须自行捕获） */
  const removePill = useCallback(async (id: string): Promise<void> => {
    try {
      await pillService.deletePill(id)
      dispatch({ type: 'REMOVE_PILL', payload: id })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '删除金丹失败' })
      throw error
    }
  }, [])

  return (
    <PillContext.Provider
      value={{
        state,
        dispatch,
        fetchPills,
        fetchPill,
        addPill,
        editPill,
        removePill,
      }}
    >
      {children}
    </PillContext.Provider>
  )
}

/** Hook */
export function usePill(): PillContextType {
  const context = useContext(PillContext)
  if (!context) {
    throw new Error('usePill must be used within a PillProvider')
  }
  return context
}
