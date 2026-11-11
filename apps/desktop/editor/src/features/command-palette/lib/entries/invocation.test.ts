/** The invocation of a command made from its target. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import { createInvocation } from './invocation'

const TARGET: CommandTarget = {
  tab: '',
  kind: null,
  path: 'Glaciers',
  title: 'Glaciers',
  file: 'Glaciers',
  isFolder: true,
  vault: { id: 'v', name: 'Harbor' },
  isReady: true,
}

describe('an invocation made from a target', () => {
  it('is marked over a named target where a place named the target', () => {
    expect(createInvocation('x', { ...TARGET, isTargetNamed: true }).target.isTargetNamed).toBe(true)
  })

  it('is over a folder where the target is one', () => {
    expect(createInvocation('x', TARGET).target.isFolder).toBe(true)
    expect(createInvocation('x', { ...TARGET, isFolder: false }).target.isFolder).toBeUndefined()
  })

  it('is not marked where the target is the tab in front', () => {
    expect(createInvocation('x', TARGET).target.isTargetNamed).toBeUndefined()
  })

  it('keeps the selector of the target, and has none where the target has none', () => {
    expect(createInvocation('x', { ...TARGET, selector: { page: 3 } }).target.selector).toEqual({
      page: 3,
    })
    expect(createInvocation('x', TARGET).target.selector).toBeUndefined()
  })

  it('keeps the line of the target, the first line included, and has none where it has none', () => {
    expect(createInvocation('x', { ...TARGET, line: 7 }).target.line).toBe(7)
    expect(createInvocation('x', { ...TARGET, line: 0 }).target.line).toBe(0)
    expect(createInvocation('x', TARGET).target.line).toBeUndefined()
  })
})
