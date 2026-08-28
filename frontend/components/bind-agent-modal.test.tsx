import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { BindAgentModal } from '@/components/bind-agent-modal'
import type { Agent, Pill } from '@/services/types'

// 真实消息解析(命名空间点路径 + {value} 插值),与 agent-card.test.tsx 同模式
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
    useTranslations: (namespace: string) => {
      const t = (key: string, values?: Record<string, unknown>) => resolveMsg(zh, namespace, key, values)
      // rich 富文本：本组件仅用 {name} 插值,函数型值(如 gold 渲染器)在无对应标签时不参与拼接
      t.rich = (key: string, values?: Record<string, unknown>) => {
        let text = resolveMsg(zh, namespace, key)
        if (values) {
          for (const [k, v] of Object.entries(values)) {
            if (typeof v !== 'function') text = text.split(`{${k}}`).join(String(v))
          }
        }
        return text
      }
      return t
    },
    useLocale: () => 'zh-CN',
  }
})

const td = vi.hoisted(() => ({
  listAgents: vi.fn(),
  bindPill: vi.fn(),
}))

vi.mock('@/services/agentService', () => ({
  listAgents: td.listAgents,
  bindPill: td.bindPill,
}))

const AGENT_1_ID = '11111111-1111-4111-8111-111111111111'
const AGENT_2_ID = '22222222-2222-4222-8222-222222222222'

const pill: Pill = {
  id: 'pill-a',
  name: '丹心妙语',
  description: '温润如茶',
  skill_schema: {},
  tags: [],
  version: '1.0.0',
  is_builtin: false,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

const agentWithAvatar: Agent = {
  id: AGENT_1_ID,
  name: '太上老君',
  avatar: 'https://example.com/laojun.png',
  personality: '沉稳如山',
  model_name: 'gpt-4o',
  status: 'active',
  proactivity: 60,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

// 无头像 fixture:继续断言首字 fallback
const agentNoAvatar: Agent = { ...agentWithAvatar, id: AGENT_2_ID, name: '沉睡道人', avatar: undefined }

describe('BindAgentModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    td.listAgents.mockResolvedValue({ list: [agentWithAvatar, agentNoAvatar], total: 2 })
    td.bindPill.mockResolvedValue(undefined)
  })
  afterEach(() => cleanup())

  it('有 avatar 的道人选项目渲染对应图片头像', async () => {
    render(<BindAgentModal pill={pill} onClose={vi.fn()} />)
    const img = await screen.findByRole('img', { name: '太上老君' })
    expect(img).toHaveAttribute('src', 'https://example.com/laojun.png')
  })

  it('无 avatar 的选项渲染首字 fallback,不创建 img', async () => {
    render(<BindAgentModal pill={pill} onClose={vi.fn()} />)
    await screen.findByText('沉睡道人')
    expect(screen.queryByRole('img', { name: '沉睡道人' })).toBeNull()
    expect(screen.getByText('沉')).toBeInTheDocument()
  })

  it('选择道人后点击服用调用 bindPill,行为不变', async () => {
    const user = userEvent.setup()
    render(<BindAgentModal pill={pill} onClose={vi.fn()} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))

    expect(td.bindPill).toHaveBeenCalledTimes(1)
    expect(td.bindPill).toHaveBeenCalledWith(AGENT_1_ID, 'pill-a', 1, 0)
  })

  it('绑定失败时展示错误,行为不变', async () => {
    const user = userEvent.setup()
    td.bindPill.mockRejectedValue(new Error('绑定失败'))
    render(<BindAgentModal pill={pill} onClose={vi.fn()} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))

    expect(await screen.findByText('绑定失败')).toBeInTheDocument()
    expect(td.bindPill).toHaveBeenCalledTimes(1)
  })
})
