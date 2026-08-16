/** What a drag away from a node comes to, worked out without a pointer. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { nodeAt, resolveDrop, seatTowards } from './drop'
import { DEFAULT_OPTIONS, resolveOptions } from './options'
import { build } from '../fixtures/build'
import type { PlexFrame, PlexRelatedSeat } from '../model'

const ALLOWED: readonly PlexRelatedSeat[] = ['parent', 'child', 'jump']

const frame = arrangePlex(build('A node', { parent: 2, child: 4, jump: 2, sibling: 2 }))
const focus = frame.nodes.find((n) => n.seat === 'focus')!
const child = frame.nodes.find((n) => n.seat === 'child')!

const drop = (from: string, at: { x: number; y: number }, allowed = ALLOWED) =>
  resolveDrop({ frame, options: DEFAULT_OPTIONS, from, at, allowed })

describe('the seat a direction stands for', () => {
  it('reads the arrangement rather than assuming which way is up', () => {
    const origin = { x: 0, y: 0 }
    const upside = resolveOptions({
      direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' },
    })

    expect(seatTowards(origin, { x: 0, y: -300 }, DEFAULT_OPTIONS)).toBe('parent')
    // Same gesture, arrangement inverted: up is where the children went.
    expect(seatTowards(origin, { x: 0, y: -300 }, upside)).toBe('child')
  })

  it('reads the axis a gesture went along, sideways being the harder ask', () => {
    const origin = { x: 0, y: 0 }
    expect(seatTowards(origin, { x: 40, y: 300 }, DEFAULT_OPTIONS)).toBe('child')
    expect(seatTowards(origin, { x: 300, y: 40 }, DEFAULT_OPTIONS)).toBe('sibling')
    expect(seatTowards(origin, { x: -300, y: 40 }, DEFAULT_OPTIONS)).toBe('jump')
  })

  it('is nothing at all when the gesture went nowhere', () => {
    expect(seatTowards({ x: 5, y: 5 }, { x: 5, y: 5 }, DEFAULT_OPTIONS)).toBeNull()
  })

  it('counts a wide row as down, though it reaches further sideways', () => {
    // The outermost child of a row is further from the focus sideways than it
    // is downwards. Splitting the space on the diagonal would name it a jump,
    // and dragging towards where the children plainly are would miss them.
    const leftmost = frame.nodes
      .filter((n) => n.seat === 'child')
      .reduce((a, b) => (a.x <= b.x ? a : b))
    expect(Math.abs(leftmost.x)).toBeGreaterThan(Math.abs(leftmost.y))
    expect(seatTowards(focus, leftmost, DEFAULT_OPTIONS)).toBe('child')

    const flat = resolveOptions({ gesture: { verticalBias: 1 } })
    expect(seatTowards(focus, leftmost, flat)).toBe('jump')
  })
})

describe('what a point lands on', () => {
  it('finds the node under it', () => {
    expect(nodeAt({ x: child.x, y: child.y }, frame)?.id).toBe(child.id)
    expect(nodeAt({ x: focus.x, y: focus.y }, frame)?.id).toBe(focus.id)
  })

  it('counts the border as inside, and a hair past it as out', () => {
    const edge = { x: child.x + child.width / 2, y: child.y }
    expect(nodeAt(edge, frame)?.id).toBe(child.id)
    expect(nodeAt({ x: edge.x + 0.001, y: edge.y }, frame)).toBeNull()
  })

  it('finds nothing in the space between', () => {
    expect(nodeAt({ x: 0, y: 4000 }, frame)).toBeNull()
  })
})

describe('letting go on nothing makes a node', () => {
  it('above the focus, a parent; below it, a child', () => {
    expect(drop(focus.id, { x: 0, y: -4000 })).toStrictEqual({
      kind: 'create',
      from: focus.id,
      seat: 'parent',
    })
    expect(drop(focus.id, { x: 0, y: 4000 })).toStrictEqual({
      kind: 'create',
      from: focus.id,
      seat: 'child',
    })
  })

  it('reads the direction from the node it started at, not from the middle', () => {
    // Straight up from a child is still above *it*, though it is below the
    // focus. A rule written against the centre would call this a child.
    const above = { x: child.x, y: child.y - 4000 }
    expect(drop(child.id, above)).toStrictEqual({
      kind: 'create',
      from: child.id,
      seat: 'parent',
    })
  })

  it('does nothing towards a seat the caller did not allow', () => {
    // A sibling is another of the parent's children — not a relationship
    // anybody can make directly — so the story leaves it out.
    expect(drop(focus.id, { x: 4000, y: 0 })).toBeNull()
    expect(drop(focus.id, { x: 4000, y: 0 }, [...ALLOWED, 'sibling'])).toStrictEqual({
      kind: 'create',
      from: focus.id,
      seat: 'sibling',
    })
  })
})

describe('letting go on a node makes a link', () => {
  it('names both ends and the seat between them', () => {
    expect(drop(focus.id, { x: child.x, y: child.y })).toStrictEqual({
      kind: 'link',
      from: focus.id,
      to: child.id,
      seat: 'child',
    })
  })

  it('takes the direction from the node landed on, not the exact point', () => {
    // Anywhere inside the box is the same landing, so a link does not change
    // its meaning depending on which corner the pointer stopped in.
    const corner = { x: child.x - child.width / 2, y: child.y + child.height / 2 }
    expect(drop(focus.id, corner)).toStrictEqual(drop(focus.id, { x: child.x, y: child.y }))
  })

  it('does nothing when it lands back where it started', () => {
    expect(drop(focus.id, { x: focus.x, y: focus.y })).toBeNull()
  })
})

describe('a gesture from a node that is not there', () => {
  it('comes to nothing', () => {
    expect(drop('nobody', { x: 0, y: 400 })).toBeNull()
  })
})

describe('the node under the pointer', () => {
  const origin = { x: 0, y: 0 }
  const stacked = (...nodes: { id: string; opacity?: number }[]): PlexFrame => ({
    ...frame,
    nodes: nodes.map((node) => ({
      seat: 'child' as const,
      title: node.id,
      x: 0,
      y: 0,
      width: 100,
      height: 40,
      order: 0,
      opacity: 1,
      ...node,
    })),
  })

  it('is the last one drawn where two sit on top of each other', () => {
    // Partway through a move a departing node and an arriving one occupy the
    // same place. The one on top is the one the reader sees, so it is the one
    // they meant.
    expect(nodeAt(origin, stacked({ id: 'under' }, { id: 'over' }))?.id).toBe('over')
  })

  it('is not one on its way in or out, however squarely it is under it', () => {
    // The rule that decides the click and the tab stop decides this too: a link
    // to something the reader never saw is not what the gesture asked for.
    expect(nodeAt(origin, stacked({ id: 'leaving', opacity: 0.4 }))).toBeNull()
  })
})
