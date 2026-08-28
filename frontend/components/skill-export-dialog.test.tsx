import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { SkillExportDialog, skillExportFilename } from '@/components/skill-export-dialog'
import { ApiError } from '@/services/api'
import type { Pill, SkillExportResult } from '@/services/types'

const exportSkill = vi.hoisted(() => vi.fn())
const saveSkillExportDesktop = vi.hoisted(() => vi.fn())
const revealSkillExport = vi.hoisted(() => vi.fn())

vi.mock('@/services/distillationService', () => ({
  exportSkill,
  saveSkillExportDesktop,
  revealSkillExport,
}))

// 键透传：断言只关心键/角色与行为，不依赖具体语言
vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

const pill: Pill = {
  id: '88888888-8888-4888-8888-888888888888',
  name: '丹心妙语',
  description: '温润如茶',
  skill_schema: {},
  tags: [],
  is_builtin: false,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
  version: '1.0.0',
}

/** 打桩浏览器下载链路(jsdom 无 createObjectURL/真实导航),返回被调用的桩 */
function stubDownload() {
  const createObjectURL = vi.fn(() => 'blob:mock-url')
  const revokeObjectURL = vi.fn()
  vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
  const anchorClick = vi
    .spyOn(HTMLAnchorElement.prototype, 'click')
    .mockImplementation(() => undefined)
  return { createObjectURL, revokeObjectURL, anchorClick }
}

const zipResult = (filename: string): SkillExportResult => ({
  blob: new Blob(['PK'], { type: 'application/zip' }),
  filename,
})

describe('SkillExportDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    document.documentElement.classList.remove('is-desktop')
  })

  it('默认 Codex：点击导出调用 exportSkill(pill_id, codex) 并触发浏览器下载', async () => {
    const { createObjectURL, anchorClick } = stubDownload()
    exportSkill.mockResolvedValue(zipResult('alchemy-skill-skill-codex.zip'))
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'exportCta' }))

    expect(exportSkill).toHaveBeenCalledWith({ pill_id: pill.id, format: 'codex' })
    // 下载：Blob 走 createObjectURL + a[download] 点击，用服务端 Content-Disposition 文件名
    await vi.waitFor(() => expect(createObjectURL).toHaveBeenCalledTimes(1))
    expect(anchorClick).toHaveBeenCalledTimes(1)
    expect(screen.getByText('success')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('切换为 Claude 后导出使用 claude 格式', async () => {
    stubDownload()
    exportSkill.mockResolvedValue(zipResult('alchemy-skill-skill-claude.zip'))
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'claude' }))
    await user.click(screen.getByRole('button', { name: 'exportCta' }))

    expect(exportSkill).toHaveBeenCalledWith({ pill_id: pill.id, format: 'claude' })
  })

  it('展示文件名并随格式切换：alchemy-skill-<slug>-<format>.zip', async () => {
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    expect(screen.getByText(skillExportFilename(pill.name, 'codex'))).toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole('button', { name: 'claude' }))
    expect(screen.getByText(skillExportFilename(pill.name, 'claude'))).toBeInTheDocument()
  })

  it('展示包含内容清单、来源说明与「不含 API Key 与网页全文」确认文案', () => {
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    expect(screen.getByText('contentsLabel')).toBeInTheDocument()
    expect(screen.getByText('contents.skillMd')).toBeInTheDocument()
    expect(screen.getByText('contents.sourcesMd')).toBeInTheDocument()
    expect(screen.getByText('contents.readmeMd')).toBeInTheDocument()
    expect(screen.queryByText('contents.platformJson')).toBeNull()
    expect(screen.getByText('sourceNote')).toBeInTheDocument()
    expect(screen.getByText('privacyNote')).toBeInTheDocument()
  })

  it('Claude 内容清单额外包含 platform/claude.json', async () => {
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'claude' }))
    expect(screen.getByText('contents.platformJson')).toBeInTheDocument()
  })

  it('导出失败：保留弹窗展示错误，点「重试」重新导出并成功下载', async () => {
    const { anchorClick } = stubDownload()
    exportSkill.mockRejectedValueOnce(new ApiError('服务器内部错误', 500))
    exportSkill.mockResolvedValueOnce(zipResult('alchemy-skill-skill-codex.zip'))
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'exportCta' }))
    // 失败：弹窗保留(标题仍在)、错误可见、重试入口可见
    expect(await screen.findByText('服务器内部错误')).toBeInTheDocument()
    expect(screen.getByText('title')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'retry' })).toBeInTheDocument()
    expect(anchorClick).not.toHaveBeenCalled()

    // 重试：同一格式重新调用 exportSkill，成功后触发下载
    await user.click(screen.getByRole('button', { name: 'retry' }))
    await vi.waitFor(() => expect(exportSkill).toHaveBeenCalledTimes(2))
    expect(exportSkill).toHaveBeenLastCalledWith({ pill_id: pill.id, format: 'codex' })
    await vi.waitFor(() => expect(anchorClick).toHaveBeenCalledTimes(1))
    expect(screen.getByText('success')).toBeInTheDocument()
  })

  // ---- 桌面 webview 场景 ----
  const savedPath = '/Users/tester/Library/Application Support/AlchemyFurnace/exports/alchemy-skill-skill-codex.zip'

  it('桌面端：导出保存到数据目录并展示路径，不触发浏览器下载', async () => {
    document.documentElement.classList.add('is-desktop')
    const { createObjectURL } = stubDownload()
    exportSkill.mockResolvedValue(zipResult('alchemy-skill-skill-codex.zip'))
    saveSkillExportDesktop.mockResolvedValue(savedPath)
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'exportCta' }))

    expect(exportSkill).toHaveBeenCalledWith({ pill_id: pill.id, format: 'codex' })
    await vi.waitFor(() => expect(saveSkillExportDesktop).toHaveBeenCalledTimes(1))
    // 落盘文件名与服务端 Content-Disposition 一致
    expect(saveSkillExportDesktop).toHaveBeenCalledWith(
      expect.any(Blob),
      'alchemy-skill-skill-codex.zip'
    )
    // 桌面端绝不走 Blob 下载
    expect(createObjectURL).not.toHaveBeenCalled()
    expect(screen.getByText('savedTo')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'openFolder' })).toBeInTheDocument()
  })

  it('桌面端：点「打开文件夹」定位已保存文件', async () => {
    document.documentElement.classList.add('is-desktop')
    exportSkill.mockResolvedValue(zipResult('alchemy-skill-skill-codex.zip'))
    saveSkillExportDesktop.mockResolvedValue(savedPath)
    revealSkillExport.mockResolvedValue(undefined)
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'exportCta' }))
    await vi.waitFor(() => expect(screen.getByRole('button', { name: 'openFolder' })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'openFolder' }))

    expect(revealSkillExport).toHaveBeenCalledWith(savedPath)
  })

  it('桌面端：保存失败保留弹窗展示错误并可重试', async () => {
    document.documentElement.classList.add('is-desktop')
    exportSkill.mockResolvedValue(zipResult('alchemy-skill-skill-codex.zip'))
    saveSkillExportDesktop.mockRejectedValueOnce(new ApiError('保存失败', 500))
    saveSkillExportDesktop.mockResolvedValueOnce(savedPath)
    const { createObjectURL } = stubDownload()
    const user = userEvent.setup()
    render(<SkillExportDialog pill={pill} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'exportCta' }))
    expect(await screen.findByText('保存失败')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'retry' })).toBeInTheDocument()
    expect(createObjectURL).not.toHaveBeenCalled()

    // 重试成功：落盘并展示路径
    await user.click(screen.getByRole('button', { name: 'retry' }))
    await vi.waitFor(() => expect(saveSkillExportDesktop).toHaveBeenCalledTimes(2))
    expect(screen.getByText('savedTo')).toBeInTheDocument()
  })
})
