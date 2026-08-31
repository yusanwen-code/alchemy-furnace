import { describe, expect, it } from 'vitest'
import { canUsePill, type PillItemState } from './pill-inventory-state'

describe('canUsePill', () => {
  it.each<[PillItemState, boolean]>([
    ['available', true],
    ['consumed_by_agent', false],
    ['consumed_by_fusion', false],
    ['discarded', false],
  ])('%s => %s', (state, expected) => {
    expect(canUsePill(state)).toBe(expected)
  })
})
