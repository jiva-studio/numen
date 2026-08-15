/** What a drag away from a node comes to, worked out without a pointer. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { nodeAt, resolveDrop, roleTowards } from './drop'
import { DEFAULT_OPTIONS, resolveOptions } from './options'
import { build } from '../fixtures/build'
import type { PlexRelatedRole } from '../model'

const ALLOWED: readonly PlexRelatedRole[] = ['parent', 'child', 'jump']

const frame = arrangePlex(build('A thought', { parent: 2, child: 4, jump: 2, sibling: 2 }))
const focus = frame.nodes.find((n) => n.role === 'focus')!
const child = frame.nodes.find((n) => n.role === 'child')!

const drop = (from: string, at: { x: number; y: number }, allowed = ALLOWED) =>
  resolveDrop({ frame, options: DEFAULT_OPTIONS, from, at, allowed })

describe('the seat a direction stands for', () => {
  it('reads the arrangement rather than assuming which way is up', () => {
    const origin = { x: 0, y: 0 }
    const upside = resolveOptions({
      direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' },
    })

    expect(roleTowards(origin, { x: 0, y: -300 }, DEFAULT_OPTIONS)).toBe('parent')
    // Same gesture, arrangement inverted: up is where the children went.
    expect(roleTowards(origin, { x: 0, y: -300 }, upside)).toBe('child')
  })

  it('takes the axis a gesture went furthest along', () => {
    const origin = { x: 0, y: 0 }
    expect(roleTowards(origin, { x: 40, y: 300 }, DEFAULT_OPTIONS)).toBe('child')
    expect(roleTowards(origin, { x: 300, y: 40 }, DEFAULT_OPTIONS)).toBe('sibling')
    expect(roleTowards(origin, { x: -300, y: 40 }, DEFAULT_OPTIONS)).toBe('jump')
  })

  it('is nothing at all when the gesture went nowhere', () => {
    expect(roleTowards({ x: 5, y: 5 }, { x: 5, y: 5 }, DEFAULT_OPTIONS)).toBeNull()
  })

  it('counts a wide row as down, though it reaches further sideways', () => {
    // The outermost child of a row is further from the focus sideways than it
    // is downwards. Splitting the space on the diagonal would name it a jump,
    // and dragging towards where the children plainly are would miss them.
    const leftmost = frame.nodes
      .filter((n) => n.role === 'child')
      .reduce((a, b) => (a.x <= b.x ? a : b))
    expect(Math.abs(leftmost.x)).toBeGreaterThan(Math.abs(leftmost.y))
    expect(roleTowards(focus, leftmost, DEFAULT_OPTIONS)).toBe('child')

    const flat = resolveOptions({ gesture: { verticalBias: 1 } })
    expect(roleTowards(focus, leftmost, flat)).toBe('jump')
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
      role: 'parent',
    })
    expect(drop(focus.id, { x: 0, y: 4000 })).toStrictEqual({
      kind: 'create',
      from: focus.id,
      role: 'child',
    })
  })

  it('reads the direction from the node it started at, not from the middle', () => {
    // Straight up from a child is still above *it*, though it is below the
    // focus. A rule written against the centre would call this a child.
    const above = { x: child.x, y: child.y - 4000 }
    expect(drop(child.id, above)).toStrictEqual({
      kind: 'create',
      from: child.id,
      role: 'parent',
    })
  })

  it('does nothing towards a seat the caller did not allow', () => {
    // A sibling is another of the parent's children — not a relationship
    // anybody can make directly — so the story leaves it out.
    expect(drop(focus.id, { x: 4000, y: 0 })).toBeNull()
    expect(drop(focus.id, { x: 4000, y: 0 }, [...ALLOWED, 'sibling'])).toStrictEqual({
      kind: 'create',
      from: focus.id,
      role: 'sibling',
    })
  })
})

describe('letting go on a node makes a link', () => {
  it('names both ends and the seat between them', () => {
    expect(drop(focus.id, { x: child.x, y: child.y })).toStrictEqual({
      kind: 'link',
      from: focus.id,
      to: child.id,
      role: 'child',
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
