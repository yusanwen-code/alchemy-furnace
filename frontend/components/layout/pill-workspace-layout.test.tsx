import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { PILL_WORKSPACE_FRAME, PillWorkspaceHeader, PillWorkspacePage } from './pill-workspace-layout'

describe('PillWorkspacePage 统一外框', () => {
  afterEach(() => cleanup())

  it('渲染有且仅有一个页面容器，携带统一宽度与上下留白', () => {
    render(
      <PillWorkspacePage>
        <p>丹方内容</p>
      </PillWorkspacePage>,
    )
    const containers = document.querySelectorAll('[data-pill-workspace-page]')
    expect(containers).toHaveLength(1)
    const container = containers[0]
    // 统一外框: 居中 + 最大宽度 + 只加一层水平留白;页面起始/底部安全间距固定
    for (const cls of PILL_WORKSPACE_FRAME.split(' ')) {
      expect(container).toHaveClass(cls)
    }
    expect(container).toHaveClass('pt-8', 'pb-24')
    expect(container).toHaveTextContent('丹方内容')
  })

  it('页头可访问: 标题/副标题/动作齐全,图标仅作装饰', () => {
    render(
      <PillWorkspacePage>
        <PillWorkspaceHeader
          icon={<span data-testid="icon" aria-hidden />}
          title="丹方"
          subtitle="按丹方炼制的全部金丹"
          actions={<button type="button">创建</button>}
        />
      </PillWorkspacePage>,
    )
    expect(screen.getByRole('heading', { name: '丹方' })).toHaveClass('page-title')
    expect(screen.getByText('按丹方炼制的全部金丹')).toHaveClass('page-subtitle')
    expect(screen.getByRole('button', { name: '创建' })).toBeInTheDocument()
    // 图标包在 aria-hidden 容器内,不进入可访问树
    expect(screen.getByTestId('icon').parentElement).toHaveAttribute('aria-hidden', 'true')
  })

  it('无 actions 时页头不渲染动作区', () => {
    render(
      <PillWorkspaceHeader icon={<span aria-hidden />} title="库存" subtitle="副标题" />,
    )
    expect(screen.getByRole('heading', { name: '库存' })).toBeInTheDocument()
  })
})
