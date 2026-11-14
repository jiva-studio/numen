/** A group of notes of different widths keeps one distance from the focus. */
import { describe, expect, it } from 'vitest'
import { clusters, getClusterDirection } from './clusters'
import { DEFAULT_OPTIONS } from '../options'
import { NO_ROOM } from './placement'
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { PlexNode } from '../../../../types'

const WIDTHS: Readonly<Record<string, number>> = { wide: 144, narrow: 72 }

const focus: PlacedNode = {
  id: 'focus',
  title: 'focus',
  relation: 'focus',
  x: 0,
  y: 0,
  width: 176,
  height: 44,
  order: 0,
  opacity: 1,
}

describe('a group of notes of different widths', () => {
  it.each(RELATION_NAMES)(
    'puts no %s note nearer the focus than the first note placed alone',
    (relation) => {
      const notes: PlexNode[] = ['wide', 'narrow', 'narrow', 'narrow'].map((width, index) => ({
        id: `${width}-${index}`,
        title: width,
        relation,
      }))
      const width = (note: PlexNode) => WIDTHS[note.title]!
      const direction = getClusterDirection(DEFAULT_OPTIONS.direction[relation], null)
      const along = (box: PlacedNode) => box.x * direction.x + box.y * direction.y

      const alone = clusters.place({ [relation]: notes.slice(0, 1) }, focus, {
        options: DEFAULT_OPTIONS,
        width,
        room: () => NO_ROOM,
      })
      const placed = clusters.place({ [relation]: notes }, focus, {
        options: DEFAULT_OPTIONS,
        width,
        room: () => NO_ROOM,
      })

      for (const box of placed) expect(along(box)).toBeGreaterThanOrEqual(along(alone[0]!) - 0.5)
    },
  )
})
