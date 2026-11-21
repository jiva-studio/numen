/** Aligning the rooms of a side keeps pills level without giving a note room under it that its views do not draw. */
import { describe, expect, it } from 'vitest'
import { DEFAULT_OPTIONS } from '../options'
import { alignRooms } from './rooms'
import type { PlexNode } from '../../../../types'

const tagged: PlexNode = { id: 'tagged', title: 'Tagged', relation: 'child' }
const bare: PlexNode = { id: 'bare', title: 'Bare', relation: 'child' }
const OWN: Record<string, { top: number; bottom: number }> = {
  tagged: { top: 30, bottom: 24 },
  bare: { top: 0, bottom: 0 },
}
const room = (node: PlexNode) => ({ ...OWN[node.id]!, width: 100 })

describe('the rooms of a side aligned to the grid', () => {
  const aligned = alignRooms({ child: [tagged, bare] }, DEFAULT_OPTIONS, room)

  it('give every note the most room over its pill, so the pills line up', () => {
    expect([aligned(tagged).top, aligned(bare).top]).toStrictEqual([30, 30])
  })

  it('leave a note whose views draw nothing under it with no room under it', () => {
    expect([aligned(tagged).bottom, aligned(bare).bottom]).toStrictEqual([24, 0])
  })

  it('reserve nothing over or under a side where no view draws', () => {
    const none = alignRooms({ child: [bare, { ...bare, id: 'other' }] }, DEFAULT_OPTIONS, () => ({
      top: 0,
      bottom: 0,
      width: 100,
    }))

    expect(none(bare)).toStrictEqual({ top: 0, bottom: 0, width: 100 })
  })
})
