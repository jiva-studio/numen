/** The invocation of a command made from its target. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '../../types'
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
    expect(createInvocation('x', { ...TARGET, isTargetNamed: true }).isTargetNamed).toBe(true)
  })

  it('is not marked where the target is the tab in front', () => {
    expect(createInvocation('x', TARGET).isTargetNamed).toBeUndefined()
  })
})
