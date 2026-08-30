import { describe, expect, it } from 'vitest'

import { avatarInputMaxLength, validateAvatarField } from '@/lib/avatar-validation'

describe('validateAvatarField', () => {
  it.each([
    ['', undefined],
    ['https://cdn.example.com/a.png', undefined],
    ['data:image/png;base64,AAAA', undefined],
    ['/a.png', 'invalid'],
    ['javascript:alert(1)', 'invalid'],
    ['https://user:pass@example.com/a.png', 'invalid'],
    ['data:image/svg+xml;base64,AAAA', 'invalid'],
    ['data:image/png;base64,@@@', 'invalid'],
  ])('validates %s', (value, expected) => {
    expect(validateAvatarField(value)).toBe(expected)
  })

  it('rejects URLs longer than 2048 characters', () => {
    const longUrl = `https://example.com/${'a'.repeat(2050)}`
    expect(validateAvatarField(longUrl)).toBe('tooLong')
  })

  it('rejects data URIs longer than 1,500,000 characters', () => {
    const longDataUri = `data:image/png;base64,${'A'.repeat(1_500_001)}`
    expect(validateAvatarField(longDataUri)).toBe('tooLong')
  })

  it('accepts jpeg/webp/gif data URIs and http URLs', () => {
    expect(validateAvatarField('data:image/jpeg;base64,AAAA')).toBeUndefined()
    expect(validateAvatarField('data:image/webp;base64,AAAA')).toBeUndefined()
    expect(validateAvatarField('data:image/gif;base64,AAAA')).toBeUndefined()
    expect(validateAvatarField('http://example.com/a.png')).toBeUndefined()
  })
})

describe('avatarInputMaxLength', () => {
  it('caps data URIs at 1,500,000 and everything else at 2048', () => {
    expect(avatarInputMaxLength('data:image/png;base64,AAAA')).toBe(1_500_000)
    expect(avatarInputMaxLength('https://example.com/a.png')).toBe(2048)
    expect(avatarInputMaxLength('')).toBe(2048)
  })
})
