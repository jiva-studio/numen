/** What a drag away from a node evaluates to, worked out without a pointer. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getDropRelation, findNode, resolveDrop, getRelationTowards } from './drop'
import { DEFAULT_OPTIONS, resolveOptions } from './options'
import { build } from '../../fixtures/build'
import type { PlexFrame } from '../frame'
import type { PlexRelation } from '../relation'

const ALLOWED: readonly PlexRelation[] = ['parent', 'child', 'jump']

const frame = arrangePlex(build('A node', { parent: 2, child: 4, jump: 2, sibling: 2 }))
const focus = frame.nodes.find((n) => n.relation === 'focus')!
const child = frame.nodes.find((n) => n.relation === 'child')!

const drop = (from: string, at: { x: number; y: number }, relations = ALLOWED) =>
  resolveDrop({ frame, options: DEFAULT_OPTIONS, from, at, relations })

describe('the relation a direction remains for', () => {
  it('reads the arrangement and not assuming which way is up', () => {
    const origin = { x: 0, y: 0 }
    const upside = resolveOptions({
      direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' },
    })

    expect(getRelationTowards(origin, { x: 0, y: -300 }, DEFAULT_OPTIONS)).toBe('parent')
    // Same gesture, arrangement inverted: up is where the children went.
    expect(getRelationTowards(origin, { x: 0, y: -300 }, upside)).toBe('child')
  })

  it('reads the axis a gesture went along, sideways being the harder ask', () => {
    const origin = { x: 0, y: 0 }
    expect(getRelationTowards(origin, { x: 40, y: 300 }, DEFAULT_OPTIONS)).toBe('child')
    expect(getRelationTowards(origin, { x: 300, y: 40 }, DEFAULT_OPTIONS)).toBe('sibling')
    expect(getRelationTowards(origin, { x: -300, y: 40 }, DEFAULT_OPTIONS)).toBe('jump')
  })

  it('counts a gesture exactly on the edge of the wedge as vertical', () => {
    const origin = { x: 0, y: 0 }
    const { verticalBias } = DEFAULT_OPTIONS.gesture
    expect(getRelationTowards(origin, { x: 10 * verticalBias, y: 10 }, DEFAULT_OPTIONS)).toBe(
      'child',
    )
    expect(getRelationTowards(origin, { x: 10 * verticalBias, y: -10 }, DEFAULT_OPTIONS)).toBe(
      'parent',
    )
    expect(getRelationTowards(origin, { x: 10 * verticalBias + 1, y: 10 }, DEFAULT_OPTIONS)).toBe(
      'sibling',
    )
  })

  it('reads a straight drag up or down as sideways to the right at a negative bias', () => {
    const origin = { x: 0, y: 0 }
    const backwards = resolveOptions({ gesture: { verticalBias: -1 } })
    expect(getRelationTowards(origin, { x: 0, y: 300 }, backwards)).toBe('sibling')
    expect(getRelationTowards(origin, { x: 0, y: -300 }, backwards)).toBe('sibling')
  })

  it('is nothing at all when the gesture went nowhere', () => {
    expect(getRelationTowards({ x: 5, y: 5 }, { x: 5, y: 5 }, DEFAULT_OPTIONS)).toBeNull()
  })

  it('counts a wide row as down, though it reaches further sideways', () => {
    // The outermost child of a row is further from the focus sideways than it
    // is downwards. Splitting the space on the diagonal would name it a jump,
    // and dragging towards where the children plainly are would miss them.
    const leftmost = frame.nodes
      .filter((n) => n.relation === 'child')
      .reduce((a, b) => (a.x <= b.x ? a : b))
    expect(Math.abs(leftmost.x)).toBeGreaterThan(Math.abs(leftmost.y))
    expect(getRelationTowards(focus, leftmost, DEFAULT_OPTIONS)).toBe('child')

    const flat = resolveOptions({ gesture: { verticalBias: 1 } })
    expect(getRelationTowards(focus, leftmost, flat)).toBe('jump')
  })
})

describe('what a point lands on', () => {
  it('finds the node under it', () => {
    expect(findNode({ x: child.x, y: child.y }, frame)?.id).toBe(child.id)
    expect(findNode({ x: focus.x, y: focus.y }, frame)?.id).toBe(focus.id)
  })

  it('counts the border as inside, and a hair past it as out', () => {
    const edge = { x: child.x + child.width / 2, y: child.y }
    expect(findNode(edge, frame)?.id).toBe(child.id)
    expect(findNode({ x: edge.x + 0.001, y: edge.y }, frame)).toBeNull()
  })

  it('finds nothing in the space between', () => {
    expect(findNode({ x: 0, y: 4000 }, frame)).toBeNull()
  })
})

describe('letting go on nothing makes a node', () => {
  it('above the focus, a parent; below it, a child', () => {
    expect(drop(focus.id, { x: 0, y: -4000 })).toStrictEqual({
      kind: 'create',
      from: focus.id,
      relation: 'parent',
    })
    expect(drop(focus.id, { x: 0, y: 4000 })).toStrictEqual({
      kind: 'create',
      from: focus.id,
      relation: 'child',
    })
  })

  it('reads the direction from the node it started at, not from the middle', () => {
    // Straight up from a child is still above *it*, though it is below the
    // focus. A rule written against the centre would call this a child.
    const above = { x: child.x, y: child.y - 4000 }
    expect(drop(child.id, above)).toStrictEqual({
      kind: 'create',
      from: child.id,
      relation: 'parent',
    })
  })

  it('does nothing towards a relation the caller did not allow', () => {
    // A sibling is another of the parent's children — not a relationship
    // anybody can make directly — so the story leaves it out.
    expect(drop(focus.id, { x: 4000, y: 0 })).toBeNull()
    expect(drop(focus.id, { x: 4000, y: 0 }, [...ALLOWED, 'sibling'])).toStrictEqual({
      kind: 'create',
      from: focus.id,
      relation: 'sibling',
    })
  })
})

describe('letting go on a node makes a link', () => {
  it('names both ends and the relation between them', () => {
    expect(drop(focus.id, { x: child.x, y: child.y })).toStrictEqual({
      kind: 'link',
      from: focus.id,
      to: child.id,
      relation: 'child',
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
  it('evaluates to nothing', () => {
    expect(drop('nobody', { x: 0, y: 400 })).toBeNull()
  })
})

describe('the node under the pointer', () => {
  const origin = { x: 0, y: 0 }
  const createFrame = (...nodes: { id: string; opacity?: number }[]): PlexFrame => ({
    ...frame,
    nodes: nodes.map((node) => ({
      relation: 'child' as const,
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
    expect(findNode(origin, createFrame({ id: 'under' }, { id: 'over' }))?.id).toBe('over')
  })

  it('is not one on its way in or out, however squarely it is under it', () => {
    // The rule that decides the click and the tab stop decides this too: a link
    // to something the reader never saw is not what the gesture asked for.
    expect(findNode(origin, createFrame({ id: 'leaving', opacity: 0.4 }))).toBeNull()
  })
})

describe('the relation something dragged in evaluates to', () => {
  const VIEWPORT = { width: 1200, height: 800 }
  const THRESHOLD = 8

  const getRelationAt = (
    at: { x: number; y: number },
    overrides: Partial<Parameters<typeof getDropRelation>[0]> = {},
  ) =>
    getDropRelation({
      frame,
      options: DEFAULT_OPTIONS,
      viewport: VIEWPORT,
      at,
      relations: ALLOWED,
      threshold: THRESHOLD,
      ...overrides,
    })

  it('is a parent above the focus, a child below it, and a jump to the side', () => {
    expect(getRelationAt({ x: focus.x, y: focus.y - 300 })).toBe('parent')
    expect(getRelationAt({ x: focus.x, y: focus.y + 300 })).toBe('child')
    expect(getRelationAt({ x: focus.x - 500, y: focus.y })).toBe('jump')
  })

  it('reads which way a relation lies off the arrangement', () => {
    const upside = resolveOptions({
      direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' },
    })
    expect(getRelationAt({ x: focus.x, y: focus.y - 300 }, { options: upside })).toBe('child')
    expect(getRelationAt({ x: focus.x, y: focus.y + 300 }, { options: upside })).toBe('parent')
  })

  it('counts a wide row as down, though it reaches further sideways', () => {
    // The outermost child of a row is further from the focus sideways than it
    // is downwards, and dragging a note to where the children plainly are has
    // to name a child.
    const leftmost = frame.nodes
      .filter((n) => n.relation === 'child')
      .reduce((a, b) => (a.x <= b.x ? a : b))
    expect(Math.abs(leftmost.x)).toBeGreaterThan(Math.abs(leftmost.y))
    expect(getRelationAt(leftmost)).toBe('child')

    const flat = resolveOptions({ gesture: { verticalBias: 1 } })
    expect(getRelationAt(leftmost, { options: flat })).toBe('jump')
  })

  it('is nothing towards a relation the caller did not allow', () => {
    const besidePoint = { x: focus.x + 500, y: focus.y }
    expect(getRelationAt(besidePoint)).toBeNull()
    expect(getRelationAt(besidePoint, { relations: [...ALLOWED, 'sibling'] })).toBe('sibling')
  })

  it('is measured from the focus, wherever the focus is drawn', () => {
    // The picture is drawn about its focus, and a rule written against the
    // middle of the window answers for a picture that never moved.
    const shifted: PlexFrame = {
      ...frame,
      nodes: frame.nodes.map((node) => ({ ...node, x: node.x + 300 })),
    }
    expect(getRelationAt({ x: 0, y: 0 }, { frame: shifted })).toBe('jump')
    expect(getRelationAt({ x: 0, y: 0 })).toBeNull()
  })

  it('is nothing until the pointer remains clear of a focus drawn off the middle', () => {
    const shifted: PlexFrame = {
      ...frame,
      nodes: frame.nodes.map((node) => ({ ...node, x: node.x + 300, y: node.y + 200 })),
    }
    const where = { x: 300, y: 200 }
    expect(getRelationAt({ x: where.x, y: where.y + THRESHOLD - 1 }, { frame: shifted })).toBeNull()
    expect(getRelationAt({ x: where.x, y: where.y + THRESHOLD }, { frame: shifted })).toBe('child')
  })

  it('is nothing past the edge of the window, and a relation on it', () => {
    expect(getRelationAt({ x: focus.x, y: 400 })).toBe('child')
    expect(getRelationAt({ x: focus.x, y: 401 })).toBeNull()
    expect(getRelationAt({ x: -600, y: focus.y })).toBe('jump')
    expect(getRelationAt({ x: -601, y: focus.y })).toBeNull()
  })

  it('is nothing until the pointer remains clear of the focus', () => {
    expect(getRelationAt({ x: focus.x, y: focus.y + THRESHOLD - 1 })).toBeNull()
    expect(getRelationAt({ x: focus.x, y: focus.y + THRESHOLD })).toBe('child')
  })

  it('is nothing where there is no focus to measure from', () => {
    const nowhere: PlexFrame = {
      ...frame,
      nodes: frame.nodes.filter((node) => node.relation !== 'focus'),
    }
    expect(getRelationAt({ x: focus.x, y: focus.y + 300 }, { frame: nowhere })).toBeNull()
  })
})
