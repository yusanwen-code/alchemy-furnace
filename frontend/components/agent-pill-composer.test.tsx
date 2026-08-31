import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgentPillComposer } from '@/components/agent-pill-composer'
import type { AgentEffect, AgentEffectDraftItem } from '@/services/types'

// 真实消息解析(命名空间点路径 + {value} 插值),与 pill-detail.test.tsx 同模式
function resolveMsg(
  messages: unknown,
  namespace: string,
  key: string,
  values?: Record<string, unknown>,
): string {
  let node: unknown = messages
  for (const part of `${namespace}.${key}`.split('.')) {
    if (node == null || typeof node !== 'object') {
      node = undefined
      break
    }
    node = (node as Record<string, unknown>)[part]
  }
  let text = typeof node === 'string' ? node : `${namespace}.${key}`
  if (values) for (const [k, v] of Object.entries(values)) text = text.split(`{${k}}`).join(String(v))
  return text
}

vi.mock('next-intl', async () => {
  const zh = (await import('@/messages/zh-CN.json')).default
  return {
    useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
      resolveMsg(zh, namespace, key, values),
    useLocale: () => 'zh-CN',
  }
})

const effA: AgentEffect = {
  id: 'eff-a',
  name: '丹心妙语',
  schema: {},
  weight: 2,
  sort_order: 1,
  item_id: 'item-a',
  revision_id: 'rev-1',
  created_at: '2026-08-20T00:00:00Z',
}
const effB: AgentEffect = { ...effA, id: 'eff-b', name: '浩然正气', weight: 1, sort_order: 2, item_id: 'item-b' }
const effC: AgentEffect = { ...effA, id: 'eff-c', name: '清风徐来', weight: 3, sort_order: 3, item_id: 'item-c' }

const twoRows: AgentEffectDraftItem[] = [
  { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
  { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
]

function renderComposer({
  value = twoRows,
  onChange = vi.fn(),
  effects = [effA, effB, effC],
  fieldErrors,
}: {
  value?: AgentEffectDraftItem[]
  onChange?: (next: AgentEffectDraftItem[]) => void
  effects?: AgentEffect[]
  fieldErrors?: Record<string, string>
} = {}) {
  return { onChange, ...render(
    <AgentPillComposer value={value} onChange={onChange} effects={effects} fieldErrors={fieldErrors} />,
  ) }
}

describe('AgentPillComposer（effects 池）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })
  afterEach(() => cleanup())

  it('按草稿顺序渲染能力名与剂量输入', () => {
    renderComposer()
    const weights = screen.getAllByRole('spinbutton')
    expect(weights).toHaveLength(2)
    expect(weights[0]).toHaveValue(2)
    expect(weights[1]).toHaveValue(1)
    expect(screen.getByText('丹心妙语')).toBeInTheDocument()
    expect(screen.getByText('浩然正气')).toBeInTheDocument()
  })

  it('剂量输入是键盘可访问的 number input,带能力名 aria-label', () => {
    renderComposer()
    expect(screen.getByRole('spinbutton', { name: /丹心妙语/ })).toBeInTheDocument()
  })

  it('修改剂量:onChange 输出整行更新的最终数组,其余行不变', () => {
    const onChange = vi.fn()
    renderComposer({ onChange })

    // 受控组件:测试不更新 value,用 fireEvent.change 直接给出目标值
    const weightInput = screen.getByRole('spinbutton', { name: /丹心妙语/ })
    fireEvent.change(weightInput, { target: { value: '5' } })
    expect(onChange).toHaveBeenCalledWith([
      { key: 'eff-a', effect_id: 'eff-a', weight: 5 },
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
    ])
  })

  it('上移/下移按钮交换顺序;首行不可上移,末行不可下移(键盘可达)', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderComposer({ onChange })

    const upFirst = screen.getByRole('button', { name: /上移 丹心妙语/ })
    const downLast = screen.getByRole('button', { name: /下移 浩然正气/ })
    expect(upFirst).toBeDisabled()
    expect(downLast).toBeDisabled()

    await user.click(screen.getByRole('button', { name: /下移 丹心妙语/ }))
    expect(onChange).toHaveBeenCalledWith([
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
      { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
    ])
  })

  it('上移第二行后其位于首位', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderComposer({ onChange })

    await user.click(screen.getByRole('button', { name: /上移 浩然正气/ }))
    expect(onChange).toHaveBeenCalledWith([
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
      { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
    ])
  })

  it('移除按钮删除对应行(能力移除不返还金丹,保存时生效)', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderComposer({ onChange })

    await user.click(screen.getByRole('button', { name: /停服 丹心妙语/ }))
    expect(onChange).toHaveBeenCalledWith([{ key: 'eff-b', effect_id: 'eff-b', weight: 1 }])
  })

  it('添加下拉只列未编排能力,选中后追加到末尾(默认剂量 1,key=effect_id)', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderComposer({ onChange })

    const select = screen.getByRole('combobox', { name: '添加金丹' })
    const options = Array.from(select.querySelectorAll('option')).map(o => o.textContent)
    expect(options).toContain('清风徐来')
    expect(options).not.toContain('丹心妙语')
    expect(options).not.toContain('浩然正气')

    await user.selectOptions(select, 'eff-c')
    expect(onChange).toHaveBeenCalledWith([
      ...twoRows,
      { key: 'eff-c', effect_id: 'eff-c', weight: 1 },
    ])
  })

  it('池中能力全部已编排时不显示添加下拉,显示提示', () => {
    renderComposer({ effects: [effA, effB] })
    expect(screen.queryByRole('combobox', { name: '添加金丹' })).toBeNull()
    expect(screen.getByText(/都已服用|all/i)).toBeInTheDocument()
  })

  it('池中找不到 effect_id 时显示未知能力占位', () => {
    renderComposer({ value: [{ key: 'gone', effect_id: 'gone', weight: 1 }] })
    expect(screen.getByText(/未知金丹/)).toBeInTheDocument()
  })

  it('fieldErrors 中 effects.<key>.weight 的机器码被翻译成错误文案展示', () => {
    renderComposer({ fieldErrors: { 'effects.eff-a.weight': 'range' } })
    expect(screen.getByRole('alert')).toHaveTextContent(/0-10/)
  })

  it('空编排时显示空提示', () => {
    renderComposer({ value: [] })
    expect(screen.getByText(/尚未服用|no pills/i)).toBeInTheDocument()
  })
})
