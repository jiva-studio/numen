/** The focus keeps the room it is given over and under its pill, like every other note, and the notes around it start outside that room. */
import { describe, expect, it } from 'vitest'
import { arrangePlex, type ArrangeInput } from './arrange'
import { NO_ROOM, type NodeRoom } from './layout/placement'
import { build } from '../../fixtures/build'
import type { PlacedNode } from '../node'
import type { PlexNode } from '../../../types'

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const arrange = (graph: ReturnType<typeof build>, room?: ArrangeInput['room']) =>
  arrangePlex(graph, { measure: byTitle, ...(room ? { room } : {}) })

const withRooms =
  (rooms: Partial<Record<PlexNode['relation'], NodeRoom>>): ArrangeInput['room'] =>
  (node) =>
    rooms[node.relation] ?? NO_ROOM

const WIDGETS_OF_FOCUS: NodeRoom = { top: 46, bottom: 72, width: 0 }

const getPillTop = (node: PlacedNode): number => node.y - node.height / 2
const getPillBottom = (node: PlacedNode): number => node.y + node.height / 2

const relatives = (frame: ReturnType<typeof arrange>, relation: PlexNode['relation']) =>
  frame.nodes.filter((node) => node.relation === relation)

const getFocus = (frame: ReturnType<typeof arrange>): PlacedNode =>
  frame.nodes.find((node) => node.relation === 'focus')!

describe('the room of the focus', () => {
  const graph = build('A note', { parent: 3, child: 7 })

  it('keeps every child below the focus by its bottom room and the top room of the child', () => {
    const rooms = withRooms({ focus: WIDGETS_OF_FOCUS, child: { top: 12, bottom: 0, width: 0 } })

    const frame = arrange(graph, rooms)

    const focus = getFocus(frame)
    const children = relatives(frame, 'child')
    expect(children).toHaveLength(7)
    for (const child of children) {
      expect(getPillTop(child) - getPillBottom(focus)).toBeGreaterThanOrEqual(72 + 12 - 1e-6)
    }
  })

  it('keeps every parent above the focus by its top room and the bottom room of the parent', () => {
    const rooms = withRooms({ focus: WIDGETS_OF_FOCUS, parent: { top: 0, bottom: 20, width: 0 } })

    const frame = arrange(graph, rooms)

    const focus = getFocus(frame)
    const parents = relatives(frame, 'parent')
    expect(parents).toHaveLength(3)
    for (const parent of parents) {
      expect(getPillTop(focus) - getPillBottom(parent)).toBeGreaterThanOrEqual(46 + 20 - 1e-6)
    }
  })

  it('leaves the parents where they are when only the bottom of the focus has room', () => {
    const without = arrange(graph)

    const frame = arrange(graph, withRooms({ focus: { top: 0, bottom: 72, width: 0 } }))

    expect(relatives(frame, 'parent').map((node) => node.y)).toStrictEqual(
      relatives(without, 'parent').map((node) => node.y),
    )
  })

  it('leaves the children where they are when only the top of the focus has room', () => {
    const without = arrange(graph)

    const frame = arrange(graph, withRooms({ focus: { top: 46, bottom: 0, width: 0 } }))

    expect(relatives(frame, 'child').map((node) => node.y)).toStrictEqual(
      relatives(without, 'child').map((node) => node.y),
    )
  })

  it('keeps the focus at the center of the frame', () => {
    const focus = getFocus(arrange(graph, withRooms({ focus: WIDGETS_OF_FOCUS })))

    expect([focus.x, focus.y]).toStrictEqual([0, 0])
  })
})

describe('the room of a related note', () => {
  const graph = build('A note', { child: 9 })

  it('pushes the next child of its column down by its bottom room', () => {
    const rooms = withRooms({ child: { top: 0, bottom: 56, width: 0 } })
    const pillHeight = getFocus(arrange(graph)).height

    const children = relatives(arrange(graph, rooms), 'child')

    for (const [index, a] of children.entries()) {
      for (const b of children.slice(index + 1)) {
        if (Math.abs(a.x - b.x) < (a.width + b.width) / 2) {
          expect(Math.abs(a.y - b.y)).toBeGreaterThanOrEqual(pillHeight + 56 - 1e-6)
        }
      }
    }
  })

  it('keeps the children clear of the room of the focus and of each other together', () => {
    const rooms = withRooms({
      focus: WIDGETS_OF_FOCUS,
      child: { top: 12, bottom: 56, width: 0 },
    })

    const frame = arrange(graph, rooms)

    const focus = getFocus(frame)
    const tops = relatives(frame, 'child').map((child) => getPillTop(child) - getPillBottom(focus))
    expect(Math.min(...tops)).toBeGreaterThanOrEqual(72 + 12 - 1e-6)
  })
})

describe('a focus with no room measured', () => {
  const graph = build('A note', { parent: 3, child: 7, sibling: 2 })

  it('keeps the spacing of a frame arranged with no room at all', () => {
    expect(arrange(graph, withRooms({}))).toStrictEqual(arrange(graph))
  })

  it('keeps the spacing around it when only the other notes have room', () => {
    const only = withRooms({ child: { top: 0, bottom: 30, width: 0 } })
    const focus = getFocus(arrange(graph, only))

    expect(focus.y).toBe(0)
    expect(relatives(arrange(graph, only), 'parent').map((node) => node.y)).toStrictEqual(
      relatives(arrange(graph), 'parent').map((node) => node.y),
    )
  })
})

describe('a room that changes', () => {
  const graph = build('A note', { child: 4 })

  it('arranges the children again, farther from the focus', () => {
    const before = relatives(
      arrange(graph, withRooms({ focus: { top: 0, bottom: 20, width: 0 } })),
      'child',
    )
    const after = relatives(
      arrange(graph, withRooms({ focus: { top: 0, bottom: 90, width: 0 } })),
      'child',
    )

    expect(Math.min(...after.map((node) => node.y))).toBeGreaterThanOrEqual(
      Math.min(...before.map((node) => node.y)) + 70 - 1e-6,
    )
  })
})
