import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PillsPage from '@/app/(main)/pills/page'
import { PillProvider } from '@/contexts/PillContext'
import type { DistillationDraft, Pill } from '@/services/types'

// ---- 可变的测试替身 ----
const td = vi.hoisted(() => ({
  listPills: vi.fn(),
  createPill: vi.fn(),
  distillNuwa: vi.fn(),
  push: vi.fn(),
}))

// 真实消息解析(命名空间点路径 + {value} 插值,与 agents page.test 一致)
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

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: td.push }),
}))

vi.mock('@/services/pillService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillService')>()
  return { ...actual, listPills: td.listPills, createPill: td.createPill }
})

// 女娲面板走真实组件(断言其渲染与 distillNuwa 调用),仅 mock 其网络依赖
vi.mock('@/services/distillationService', () => ({
  distillNuwa: td.distillNuwa,
}))

// 卡片/弹窗行为在各自测试覆盖,此处隔离
vi.mock('@/components/pill-card', () => ({
  PillCard: () => <div data-testid="pill-card" />,
}))

vi.mock('@/components/bind-agent-modal', () => ({
  BindAgentModal: () => null,
}))

// TopTabs 依赖布局测量,替换为简单按钮组
vi.mock('@/components/interaction/top-tabs', () => ({
  TopTabs: ({
    tabs,
    activeKey,
    onChange,
  }: {
    tabs: Array<{ key: string; label: string }>
    activeKey: string
    onChange: (key: string) => void
  }) => (
    <div role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          type="button"
          aria-pressed={activeKey === tab.key}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  ),
}))

const pill: Pill = {
  id: 'pill-1',
  name: '丹心妙语',
  description: '温润如茶',
  skill_schema: {},
  tags: [],
  version: '1.0.0',
  is_builtin: false,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

const nuwaDraft: DistillationDraft = {
  name: '女娲草稿',
  description: '由女娲蒸馏出的候选',
  persona_summary: '冷静克制的史官',
  tags: ['神话', '蒸馏'],
  skill_schema: { identity_card: '史官' },
  sources: [
    { title: '史记', url: 'https://example.com/shiji', dimension: 'tone' },
  ],
  model: 'gpt-5',
  research: {
    evidence_level: 'standard',
    document_count: 1,
    domain_count: 1,
    total_characters: 2000,
    warnings: [],
  },
}

function renderPage() {
  return render(
    <PillProvider>
      <PillsPage />
    </PillProvider>,
  )
}

describe('PillsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    td.listPills.mockResolvedValue({ list: [pill], total: 1, page: 1, page_size: 100 })
    td.distillNuwa.mockResolvedValue(nuwaDraft)
  })
  afterEach(() => cleanup())

  it('创建金丹弹窗渲染女娲面板,且仅由用户点击触发一次 distillNuwa', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByTestId('pill-card')

    await user.click(screen.getByRole('button', { name: '炼制新金丹' }))
    // 弹窗打开即渲染唯一女娲入口
    expect(screen.getAllByText('女娲智能蒸馏')).toHaveLength(1)
    // 打开弹窗不自动触发蒸馏
    expect(td.distillNuwa).not.toHaveBeenCalled()

    // 用户显式触发一次蒸馏
    await user.type(screen.getByPlaceholderText(/保罗·格雷厄姆/), '保罗·格雷厄姆')
    await user.type(screen.getByPlaceholderText(/思考方式/), '提取他的判断方式')
    await user.click(screen.getByRole('button', { name: '从互联网收集并蒸馏' }))

    await waitFor(() => expect(td.distillNuwa).toHaveBeenCalledTimes(1))
    expect(td.distillNuwa).toHaveBeenCalledWith({
      subject: '保罗·格雷厄姆',
      brief: '提取他的判断方式',
      locale: 'zh-CN',
    })
  })
})
