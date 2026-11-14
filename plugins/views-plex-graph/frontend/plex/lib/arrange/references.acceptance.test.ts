/** References and jumps share the block left of the focus, the references over the jumps. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getBox, isOverlapping } from './layout/boxes'
import { build } from '../../fixtures/build'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexGraph, PlexNodeRelation, PlexRelation } from '../../../types'

type Window = { width: number; height: number }

const arrange = (graph: PlexGraph, viewport?: Window): PlexFrame =>
  arrangePlex(graph, viewport ? { options: { viewport } } : {})

const getFocus = (frame: PlexFrame): PlacedNode => frame.nodes.find((n) => n.relation === 'focus')!
const getShownNodes = (frame: PlexFrame, relation: PlexNodeRelation): PlacedNode[] =>
  frame.nodes.filter((node) => node.relation === relation)
const mean = (values: readonly number[]): number =>
  values.reduce((sum, value) => sum + value, 0) / values.length

const getOverlaps = (frame: PlexFrame): string[] =>
  frame.nodes.flatMap((a, index) =>
    frame.nodes
      .slice(index + 1)
      .filter((b) => isOverlapping(getBox(a, 0), getBox(b, 0)))
      .map((b) => `${a.id} / ${b.id}`),
  )

describe('references and jumps in a window', () => {
  const frame = arrange(build('A note', { reference: 6, jump: 6 }), { width: 1600, height: 1000 })
  const focus = getFocus(frame)
  const references = getShownNodes(frame, 'reference')
  const jumps = getShownNodes(frame, 'jump')

  it('shows every reference and every jump', () => {
    expect(references).toHaveLength(6)
    expect(jumps).toHaveLength(6)
  })

  it('puts the references and the jumps left of the focus', () => {
    const left = focus.x - focus.width / 2
    for (const node of [...references, ...jumps]) expect(node.x + node.width / 2).toBeLessThan(left)
  })

  it('puts the references above the focus and the jumps below it', () => {
    expect(mean(references.map((node) => node.y))).toBeLessThan(focus.y)
    expect(mean(jumps.map((node) => node.y))).toBeGreaterThan(focus.y)
  })

  it('overlaps no two boxes', () => {
    expect(getOverlaps(frame)).toStrictEqual([])
  })
})

describe('all five relations together', () => {
  const FIVE = { parent: 3, child: 8, jump: 4, reference: 5, sibling: 3 }
  const five = build('A note', FIVE)
  const WINDOWS: readonly (Window | undefined)[] = [
    undefined,
    { width: 1200, height: 800 },
    { width: 700, height: 600 },
    { width: 420, height: 320 },
  ]

  it.each(WINDOWS)('overlaps no two boxes in window %j', (viewport) => {
    expect(getOverlaps(arrange(five, viewport))).toStrictEqual([])
  })

  it.each(WINDOWS)('counts the references shown and left out for window %j', (viewport) => {
    const many = build('A note', { ...FIVE, reference: 40 })
    const frame = arrange(many, viewport)

    expect(getShownNodes(frame, 'reference').length + (frame.overflow.reference ?? 0)).toBe(40)
  })

  it('leaves some references out of a small window and says how many', () => {
    const frame = arrange(build('A note', { ...FIVE, reference: 40 }), { width: 420, height: 320 })

    expect(frame.overflow.reference).toBeGreaterThan(0)
  })

  it('places a reference from each relation of the five by name', () => {
    const relations = new Set<PlexRelation>(
      arrange(build('A note', FIVE), { width: 1600, height: 1000 })
        .nodes.filter((node) => node.relation !== 'focus')
        .map((node) => node.relation as PlexRelation),
    )

    expect([...relations].sort()).toEqual(['child', 'jump', 'parent', 'reference', 'sibling'])
  })
})
