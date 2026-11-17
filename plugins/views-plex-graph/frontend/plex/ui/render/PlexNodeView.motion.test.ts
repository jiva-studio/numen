/** A note that moves between two arrangements keeps the title it settles with: the room its title is laid out in does not change from frame to frame. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import { interpolatePlex } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import type { PlacedNode } from '../../lib/node'

const REST_HEIGHT = 36

const createNode = (overrides: Partial<PlacedNode>): PlacedNode => ({
  id: 'a',
  title: 'A title that is longer than the box it ends in',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: REST_HEIGHT,
  order: 0,
  opacity: 1,
  ...overrides,
})

const createFrame = (nodes: PlacedNode[]): PlexFrame => ({
  nodes,
  edges: [],
  extent: { minX: -200, minY: -100, maxX: 200, maxY: 100 },
  overflow: {},
})

/** The room the title text is laid out in, in the units the text is drawn in. */
const readTitleRoom = (node: PlacedNode): number => {
  const view = mount(PlexNodeView, { props: { node, restHeight: REST_HEIGHT } })
  return Number(view.get('foreignObject').attributes('width'))
}

const find = (frame: PlexFrame, id: string): PlacedNode => frame.nodes.find((n) => n.id === id)!

const MOMENTS = [0.1, 0.25, 0.5, 0.75, 0.9]

describe('the room of a title while its note moves', () => {
  it('stays the room of the settled arrangement for a note that is in both', () => {
    const from = createFrame([createNode({ width: 144, height: 36 })])
    const to = createFrame([
      createNode({ relation: 'parent', width: 90, height: 36, x: 0, y: -80 }),
    ])
    const settled = readTitleRoom(find(to, 'a'))

    for (const t of MOMENTS) {
      expect(readTitleRoom(find(interpolatePlex(from, to, t), 'a')), `at ${t}`).toBeCloseTo(settled)
    }
  })

  it('stays the room of the settled arrangement for a note that is in focus at the end', () => {
    const from = createFrame([createNode({ width: 144, height: 36 })])
    const to = createFrame([createNode({ relation: 'focus', width: 150, height: 44 })])
    const settled = readTitleRoom(find(to, 'a'))

    for (const t of MOMENTS) {
      expect(readTitleRoom(find(interpolatePlex(from, to, t), 'a')), `at ${t}`).toBeCloseTo(settled)
    }
  })

  it('is the final room for a note that enters from a narrower box', () => {
    const from = createFrame([createNode({ id: 'chosen', width: 60 })])
    const to = createFrame([
      createNode({ id: 'chosen', relation: 'focus', width: 150, height: 44 }),
      createNode({ id: 'new', width: 144, y: 80 }),
    ])
    const settled = readTitleRoom(find(to, 'new'))

    for (const t of MOMENTS) {
      expect(readTitleRoom(find(interpolatePlex(from, to, t), 'new')), `at ${t}`).toBeCloseTo(
        settled,
      )
    }
  })

  it('is the old room for a note that leaves', () => {
    const from = createFrame([
      createNode({ id: 'chosen', width: 60 }),
      createNode({ id: 'gone', width: 144, y: 80 }),
    ])
    const to = createFrame([createNode({ id: 'chosen', relation: 'focus', width: 150, height: 44 })])
    const settled = readTitleRoom(find(from, 'gone'))

    for (const t of MOMENTS) {
      expect(readTitleRoom(find(interpolatePlex(from, to, t), 'gone')), `at ${t}`).toBeCloseTo(
        settled,
      )
    }
  })
})
