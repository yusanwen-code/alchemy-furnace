import { describe, expect, it } from 'vitest'
import {
  agentDetailHref,
  pillDetailHref,
  parseEntityDetailId,
  parseLegacyEntityDetailPath,
} from '@/lib/entity-detail-route'

const id = '11111111-1111-4111-8111-111111111111'

describe('entity detail route contract', () => {
  it('builds canonical static detail URLs', () => {
    expect(agentDetailHref(id)).toBe(`/agents/detail?id=${id}`)
    expect(pillDetailHref(id)).toBe(`/pills/detail?id=${id}`)
  })

  it('rejects invalid ids when building URLs', () => {
    expect(() => agentDetailHref('_')).toThrow('Invalid entity id')
    expect(() => pillDetailHref('pill-1')).toThrow('Invalid entity id')
  })

  it('parses only a valid UUID query value', () => {
    expect(parseEntityDetailId(new URLSearchParams(`id=${id}`))).toBe(id)
    expect(parseEntityDetailId(new URLSearchParams())).toBeUndefined()
    expect(parseEntityDetailId(new URLSearchParams('id=_'))).toBeUndefined()
    expect(parseEntityDetailId(new URLSearchParams('id=bad'))).toBeUndefined()
  })

  it('recognizes only historical agent and pill UUID paths', () => {
    expect(parseLegacyEntityDetailPath(`/agents/${id}`)).toEqual({ kind: 'agents', id })
    expect(parseLegacyEntityDetailPath(`/pills/${id}/`)).toEqual({ kind: 'pills', id })
    expect(parseLegacyEntityDetailPath('/agents/detail')).toBeUndefined()
    expect(parseLegacyEntityDetailPath('/chat/' + id)).toBeUndefined()
  })
})
