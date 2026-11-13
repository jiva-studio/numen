/** The relation a drag means is chosen among the relations the caller allows, not by the order of the relation table. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { DEFAULT_OPTIONS, resolveOptions } from '../options'
import { build } from '../../../fixtures/build'
import { getRelationTowards, resolveDrop } from './drop'

const origin = { x: 0, y: 0 }
const left = { x: -300, y: 0 }

describe('the relation a drag means among the allowed ones', () => {
  it('keeps a leftward drag a jump when jump is allowed and reference is not', () => {
    expect(getRelationTowards(origin, left, DEFAULT_OPTIONS, ['parent', 'child', 'jump'])).toBe(
      'jump',
    )
  })

  it('never turns a leftward drag into a reference when only jump is allowed', () => {
    const frame = arrangePlex(build('A node', { child: 2, jump: 2 }))
    const focus = frame.nodes.find((n) => n.relation === 'focus')!
    const drop = resolveDrop({
      frame,
      options: DEFAULT_OPTIONS,
      from: focus.id,
      point: { x: focus.x - 400, y: focus.y },
      relations: ['jump'],
    })
    expect(drop).toEqual({ kind: 'create', from: focus.id, relation: 'jump' })
  })

  it('picks the first allowed relation of a heading two relations share, in any table order', () => {
    const shared = resolveOptions({
      direction: { parent: 'down', child: 'up', jump: 'up', reference: 'left', sibling: 'right' },
    })
    const up = { x: 0, y: -300 }
    expect(getRelationTowards(origin, up, shared, ['jump'])).toBe('jump')
    expect(getRelationTowards(origin, up, shared, ['jump', 'child'])).toBe('jump')
    expect(getRelationTowards(origin, up, shared, ['child', 'jump'])).toBe('child')
  })

  it('answers nothing when no allowed relation lies that way', () => {
    expect(getRelationTowards(origin, left, DEFAULT_OPTIONS, ['parent'])).toBeNull()
  })
})
