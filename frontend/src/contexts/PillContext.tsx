/**
 * 金丹状态管理 Context
 * 使用 React Context + useReducer 管理金丹相关状态
 * 包含金丹列表、当前选中的金丹、加载状态等
 */
import React, { createContext, useContext, useReducer, useCallback } from 'react'
import * as pillService from '@/services/pillService'
import type { Pill, Recipe } from '@/services/types'

/** 金丹状态 */
interface PillState {
  pills: Pill[]
  currentPill: Pill | null
  currentRecipes: Recipe[]
  loading: boolean
  error: string | null
}

/** 操作类型 */
type PillAction =
  | { type: 'SET_PILLS'; payload: Pill[] }
  | { type: 'SET_CURRENT_PILL'; payload: Pill | null }
  | { type: 'SET_CURRENT_RECIPES'; payload: Recipe[] }
  | { type: 'ADD_PILL'; payload: Pill }
  | { type: 'UPDATE_PILL'; payload: Pill }
  | { type: 'REMOVE_PILL'; payload: number }
  | { type: 'ADD_RECIPE'; payload: Recipe }
  | { type: 'REMOVE_RECIPE'; payload: number }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }

/** 初始状态 */
const initialState: PillState = {
  pills: [],
  currentPill: null,
  currentRecipes: [],
  loading: false,
  error: null,
}

/** Reducer */
function pillReducer(state: PillState, action: PillAction): PillState {
  switch (action.type) {
    case 'SET_PILLS':
      return { ...state, pills: action.payload, loading: false }
    case 'SET_CURRENT_PILL':
      return { ...state, currentPill: action.payload }
    case 'SET_CURRENT_RECIPES':
      return { ...state, currentRecipes: action.payload }
    case 'ADD_PILL':
      return { ...state, pills: [action.payload, ...state.pills] }
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
    case 'ADD_RECIPE':
      return { ...state, currentRecipes: [...state.currentRecipes, action.payload] }
    case 'REMOVE_RECIPE':
      return {
        ...state,
        currentRecipes: state.currentRecipes.filter(r => r.id !== action.payload),
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
interface PillContextType {
  state: PillState
  dispatch: React.Dispatch<PillAction>
  // 异步操作
  fetchPills: () => Promise<void>
  fetchPill: (id: number) => Promise<void>
  fetchRecipes: (pillId: number) => Promise<void>
  addPill: (name: string, description?: string) => Promise<void>
  removePill: (id: number) => Promise<void>
  uploadRecipes: (pillId: number, files: FileList) => Promise<void>
  removeRecipe: (id: number) => Promise<void>
  reExtractRecipe: (id: number) => Promise<void>
}

const PillContext = createContext<PillContextType | null>(null)

/** Provider 组件 */
export function PillProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(pillReducer, initialState)

  /** 获取金丹列表 */
  const fetchPills = useCallback(async () => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const pills = await pillService.getPills()
      dispatch({ type: 'SET_PILLS', payload: pills })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取金丹列表失败' })
    }
  }, [])

  /** 获取单个金丹 */
  const fetchPill = useCallback(async (id: number) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const pill = await pillService.getPill(id)
      dispatch({ type: 'SET_CURRENT_PILL', payload: pill })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取金丹详情失败' })
    }
  }, [])

  /** 获取丹方列表 */
  const fetchRecipes = useCallback(async (pillId: number) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const recipes = await pillService.getRecipesByPill(pillId)
      dispatch({ type: 'SET_CURRENT_RECIPES', payload: recipes })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '获取丹方列表失败' })
    }
  }, [])

  /** 创建金丹 */
  const addPill = useCallback(async (name: string, description?: string) => {
    dispatch({ type: 'SET_LOADING', payload: true })
    try {
      const pill = await pillService.createPill({ name, description })
      dispatch({ type: 'ADD_PILL', payload: pill })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '创建金丹失败' })
    }
  }, [])

  /** 删除金丹 */
  const removePill = useCallback(async (id: number) => {
    try {
      await pillService.deletePill(id)
      dispatch({ type: 'REMOVE_PILL', payload: id })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '删除金丹失败' })
    }
  }, [])

  /** 上传丹方 */
  const uploadRecipes = useCallback(async (pillId: number, files: FileList) => {
    try {
      await pillService.uploadRecipes(pillId, files)
      // 刷新丹方列表
      const recipes = await pillService.getRecipesByPill(pillId)
      dispatch({ type: 'SET_CURRENT_RECIPES', payload: recipes })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '上传丹方失败' })
    }
  }, [])

  /** 删除丹方 */
  const removeRecipe = useCallback(async (id: number) => {
    try {
      await pillService.deleteRecipe(id)
      dispatch({ type: 'REMOVE_RECIPE', payload: id })
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '删除丹方失败' })
    }
  }, [])

  /** 重新提取 */
  const reExtractRecipe = useCallback(async (id: number) => {
    try {
      await pillService.reExtractRecipe(id)
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : '重新提取失败' })
    }
  }, [])

  return (
    <PillContext.Provider
      value={{
        state,
        dispatch,
        fetchPills,
        fetchPill,
        fetchRecipes,
        addPill,
        removePill,
        uploadRecipes,
        removeRecipe,
        reExtractRecipe,
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
