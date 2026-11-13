/** What the final layout is held to: it lies inside the window, or the notes outside are counted as overflow; a measure that is not a width is not used. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { RELATION_NAMES } from '../relation'
import { applyRoom } from './look'
import { resolveOptions } from './options'
import type { Placement, PlacementContext } from './layout/placement'
import type { PlacedNode } from '../node'
import type { PlexGraph } from '../../../types'

const WINDOW = { width: 600, height: 600 }
const FAR = 10_000

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Focus', relation: 'focus' },
    { id: 'child-0', title: 'First', relation: 'child' },
    { id: 'child-1', title: 'Second', relation: 'child' },
    { id: 'child-2', title: 'Third', relation: 'child' },
  ],
  edges: [],
}

const baseline = applyRoom(resolveOptions({ viewport: WINDOW }))

/** One note per row; `isMisplaced` sends a note far outside the window. */
function createRows(
  isMisplaced: (context: PlacementContext, node: PlacedNode) => boolean,
  seen: PlacementContext[] = [],
): Placement {
  return {
    name: 'rows',
    place(nodesByRelation, _focus, context) {
      seen.push(context)
      const placed: PlacedNode[] = []
      for (const relation of RELATION_NAMES) {
        for (const node of nodesByRelation[relation] ?? []) {
          const height = context.options.nodeSize.height
          const row = placed.length + 1
          const box: PlacedNode = {
            ...node,
            x: 0,
            y: row * (height + context.options.lineGap),
            width: context.width(node),
            height,
            order: row,
            opacity: 1,
          }
          placed.push(isMisplaced(context, box) ? { ...box, y: FAR } : box)
          if (context.shouldStop?.(box)) return placed
        }
      }
      return placed
    },
  }
}

const getShown = (frame: ReturnType<typeof arrangePlex>) =>
  frame.nodes.filter((node) => node.relation !== 'focus')

describe('a final layout that leaves the window', () => {
  it('falls back to the layout admission verified', () => {
    const seen: PlacementContext[] = []
    const placement = createRows(
      (context) => context.fits !== undefined && context.options.lineGap > baseline.lineGap,
      seen,
    )

    const frame = arrangePlex(graph, { options: { viewport: WINDOW }, placement })

    const finals = seen.filter((context) => context.fits && !context.shouldStop)
    expect(finals[0]?.options.lineGap).toBeGreaterThan(baseline.lineGap)
    expect(getShown(frame).map((node) => node.y)).toStrictEqual(
      [1, 2, 3].map((row) => row * (baseline.nodeSize.height + baseline.lineGap)),
    )
    expect(frame.overflow).toStrictEqual({})
  })

  it('counts a note that is outside in both layouts as overflow', () => {
    const placement = createRows(
      (context, node) => !context.shouldStop && context.fits !== undefined && node.id === 'child-1',
    )

    const frame = arrangePlex(graph, { options: { viewport: WINDOW }, placement })

    expect(getShown(frame).map((node) => node.id)).toStrictEqual(['child-0', 'child-2'])
    expect(frame.overflow).toStrictEqual({ child: 1 })
    expect(Math.max(...frame.nodes.map((node) => node.y))).toBeLessThan(WINDOW.height)
  })

  it('leaves a layout inside the window as it is', () => {
    const frame = arrangePlex(graph, { options: { viewport: WINDOW }, placement: createRows(() => false) })

    expect(getShown(frame)).toHaveLength(3)
    expect(frame.overflow).toStrictEqual({})
  })
})

describe('a measure that is not a width', () => {
  it.each([0, -20])('gives a box the least width: %s', (measured) => {
    const frame = arrangePlex(graph, { measure: (node) => (node.id === 'child-1' ? measured : 120) })

    expect(frame.nodes.find((node) => node.id === 'child-1')!.width).toBe(baseline.minWidth)
  })

  const widthOf = (frame: ReturnType<typeof arrangePlex>, id: string) =>
    frame.nodes.find((node) => node.id === id)!.width

  it.each([Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY])(
    'is taken as not measured: %s',
    (measured) => {
      const frame = arrangePlex(graph, {
        options: { viewport: WINDOW },
        measure: (node) => (node.id === 'child-1' ? measured : 120),
      })

      expect(widthOf(frame, 'child-1')).toBe(baseline.nodeSize.width)
      for (const node of frame.nodes) {
        expect(Number.isFinite(node.x) && Number.isFinite(node.y) && Number.isFinite(node.width)).toBe(true)
      }
      expect(Object.values(frame.extent).every(Number.isFinite)).toBe(true)
    },
  )
})
