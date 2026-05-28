/**
 * 金丹服务 - 知识库管理 API
 * 提供金丹的增删改查操作（演示模式使用 Mock 数据）
 */
import { mockDelay } from './api'
import { mockPills, mockRecipes } from './mockData'
import type { Pill, Recipe, CreatePillRequest } from './types'

let pills = [...mockPills]
let recipes = { ...mockRecipes }
let nextPillId = Math.max(...pills.map(p => p.id)) + 1
let nextRecipeId = Math.max(...Object.values(recipes).flat().map(r => r.id)) + 1

/**
 * 获取金丹列表
 */
export async function getPills(): Promise<Pill[]> {
  await mockDelay()
  return [...pills].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
}

/**
 * 获取单个金丹详情
 */
export async function getPill(id: number): Promise<Pill> {
  await mockDelay()
  const pill = pills.find(p => p.id === id)
  if (!pill) throw new Error('金丹不存在')
  return { ...pill }
}

/**
 * 创建金丹
 */
export async function createPill(data: CreatePillRequest): Promise<Pill> {
  await mockDelay(600)
  const pill: Pill = {
    id: nextPillId++,
    name: data.name,
    description: data.description,
    status: 'refining',
    vector_count: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
  pills.push(pill)
  recipes[pill.id] = []
  return { ...pill }
}

/**
 * 更新金丹
 */
export async function updatePill(id: number, data: Partial<CreatePillRequest>): Promise<Pill> {
  await mockDelay()
  const index = pills.findIndex(p => p.id === id)
  if (index === -1) throw new Error('金丹不存在')
  pills[index] = {
    ...pills[index],
    ...data,
    updated_at: new Date().toISOString(),
  }
  return { ...pills[index] }
}

/**
 * 删除金丹（级联删除丹方）
 */
export async function deletePill(id: number): Promise<void> {
  await mockDelay()
  pills = pills.filter(p => p.id !== id)
  delete recipes[id]
}

/**
 * 获取金丹下的丹方列表
 */
export async function getRecipesByPill(pillId: number): Promise<Recipe[]> {
  await mockDelay()
  return [...(recipes[pillId] || [])]
}

/**
 * 上传丹方文件
 */
export async function uploadRecipes(pillId: number, files: FileList): Promise<Recipe[]> {
  await mockDelay(1000)
  const newRecipes: Recipe[] = []
  for (const file of Array.from(files)) {
    const ext = file.name.split('.').pop()?.toLowerCase() || 'txt'
    const recipe: Recipe = {
      id: nextRecipeId++,
      pill_id: pillId,
      filename: file.name,
      file_type: ext,
      file_size: file.size,
      file_path: `/uploads/${pillId}/${file.name}`,
      extract_status: 'pending',
      chunk_count: 0,
      created_at: new Date().toISOString(),
    }
    newRecipes.push(recipe)
  }
  if (!recipes[pillId]) recipes[pillId] = []
  recipes[pillId].push(...newRecipes)

  // 更新金丹的丹方数量和向量数
  const pillIndex = pills.findIndex(p => p.id === pillId)
  if (pillIndex !== -1) {
    pills[pillIndex] = {
      ...pills[pillIndex],
      updated_at: new Date().toISOString(),
    }
  }

  return newRecipes
}

/**
 * 删除丹方
 */
export async function deleteRecipe(id: number): Promise<void> {
  await mockDelay()
  for (const pillId of Object.keys(recipes)) {
    recipes[Number(pillId)] = recipes[Number(pillId)].filter(r => r.id !== id)
  }
}

/**
 * 重新提取丹方
 */
export async function reExtractRecipe(id: number): Promise<void> {
  await mockDelay(800)
  for (const pillId of Object.keys(recipes)) {
    const recipe = recipes[Number(pillId)].find(r => r.id === id)
    if (recipe) {
      recipe.extract_status = 'extracting'
      // 模拟异步完成
      setTimeout(() => {
        recipe.extract_status = 'completed'
        recipe.chunk_count = Math.floor(Math.random() * 200) + 50
      }, 2000)
      break
    }
  }
}
