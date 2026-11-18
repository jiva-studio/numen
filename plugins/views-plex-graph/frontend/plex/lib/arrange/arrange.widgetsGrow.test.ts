/** A note is never left outside the window by widgets that grew after the notes were chosen, and the notes cut are counted. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getFootprint, getFootprintCenterY } from './layout/placement'
import type { PlexGraph, PlexNode } from '../../../types'

const VIEWPORT = { width: 1000, height: 600 }
const TAG_ROW = 24
const NAME_LINE = 18
const WIDTH = 120

function createGraph(): PlexGraph {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Gears case 15', relation: 'focus' }]
  for (let index = 0; index < 20; index++) {
    nodes.push({ id: `child-${index}`, title: `Gear ${index}`, relation: 'child' })
  }
  return { nodes, edges: nodes.slice(1).map((node) => ({ from: 'focus', to: node.id })) }
}

const roomOf = (tagRows: number, nameLines: number) => (node: PlexNode) =>
  node.relation === 'focus'
    ? { top: 0, bottom: 0, width: WIDTH }
    : { top: nameLines * NAME_LINE, bottom: tagRows * TAG_ROW, width: WIDTH }

const findOutside = (frame: ReturnType<typeof arrangePlex>, room: ReturnType<typeof roomOf>) =>
  frame.nodes
    .filter((node) => node.relation !== 'focus')
    .filter((node) => {
      const footprint = getFootprint(node, room(node))
      const y = getFootprintCenterY(node.y, room(node))
      return (
        Math.abs(node.x) + footprint.width / 2 > VIEWPORT.width / 2 ||
        Math.abs(y) + footprint.height / 2 > VIEWPORT.height / 2
      )
    })
    .map((node) => node.id)

describe('a focus with twenty children whose widgets grow after they were shown', () => {
  const graph = createGraph()
  const given = { options: { viewport: VIEWPORT }, measure: () => WIDTH }
  const small = roomOf(1, 0)
  const grown = roomOf(2, 2)
  const candidates = new Set(
    arrangePlex(graph, { ...given, room: undefined }).nodes.map((n) => n.id),
  )
  const first = arrangePlex(graph, { ...given, room: small, only: candidates })
  const shown = new Set(first.nodes.map((node) => node.id))
  const again = arrangePlex(graph, { ...given, room: grown, only: candidates, keep: shown })

  it('cuts children when the widgets have grown', () => {
    expect(again.nodes.length - 1).toBeLessThan(20)
  })

  it('leaves no shown note outside the window', () => {
    expect(findOutside(again, grown)).toStrictEqual([])
  })

  it('counts every child that is not shown', () => {
    const children = again.nodes.filter((node) => node.relation === 'child').length

    expect(again.overflow.child ?? 0).toBe(20 - children)
  })

  it('shows only notes the former arrangement showed', () => {
    expect(again.nodes.every((node) => shown.has(node.id))).toBe(true)
  })
})
