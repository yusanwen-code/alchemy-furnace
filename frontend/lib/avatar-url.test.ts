import { describe, expect, it } from 'vitest'

import { normalizeAvatarUrl } from '@/lib/avatar-url'

describe('normalizeAvatarUrl', () => {
  it('accepts http/https and image data URIs', () => {
    expect(normalizeAvatarUrl(' https://cdn.example.com/a.png ')).toBe('https://cdn.example.com/a.png')
    expect(normalizeAvatarUrl('data:image/png;base64,AAAA')).toBe('data:image/png;base64,AAAA')
  })

  it('accepts plain http URLs', () => {
    expect(normalizeAvatarUrl('http://example.com/avatar.jpg')).toBe('http://example.com/avatar.jpg')
  })

  it('rejects blank, relative and executable schemes', () => {
    expect(normalizeAvatarUrl('')).toBeUndefined()
    expect(normalizeAvatarUrl('/avatar.png')).toBeUndefined()
    expect(normalizeAvatarUrl('javascript:alert(1)')).toBeUndefined()
    expect(normalizeAvatarUrl('blob:https://example.com/x')).toBeUndefined()
  })

  it('rejects whitespace-only, non-string values and other schemes', () => {
    expect(normalizeAvatarUrl('   ')).toBeUndefined()
    expect(normalizeAvatarUrl(undefined)).toBeUndefined()
    expect(normalizeAvatarUrl(null)).toBeUndefined()
    expect(normalizeAvatarUrl(123)).toBeUndefined()
    expect(normalizeAvatarUrl('vbscript:msgbox(1)')).toBeUndefined()
    expect(normalizeAvatarUrl('avatar.png')).toBeUndefined()
  })
})
