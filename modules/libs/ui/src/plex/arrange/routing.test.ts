/** How a line is drawn between two boxes. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import type { PlacedNode, PlexRole } from '../model'

const withRole = (frame: { nodes: readonly PlacedNode[] }, role: PlexRole) =>
  frame.nodes.filter((node) => node.role === role)

const focusOf = (frame: { nodes: readonly PlacedNode[] }) => {
  const focus = frame.nodes.find((node) => node.role === 'focus')
  if (!focus) throw new Error('unreachable: every frame places a focus')
  return focus
}
describe('edges', () => {
  it('leaves the focus by one gate and arrives at the top of each child', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = focusOf(layout)
    const children = withRole(layout, 'child')

    for (const child of children) {
      const edge = layout.edges.find((e) => e.to === child.id)
      expect(edge, `no edge to ${child.id}`).toBeDefined()

      // Every child hangs from the same point under the focus...
      expect(edge!.fromPoint).toStrictEqual({ x: focus.x, y: focus.y + focus.height / 2 })
      // ...and meets its own box square on top, however far sideways it sits.
      expect(edge!.toPoint).toStrictEqual({ x: child.x, y: child.y - child.height / 2 })
    }
  })

  it('sets off and arrives along the axis, never across it', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const outermost = withRole(layout, 'child').reduce((a, b) =>
      Math.abs(a.x) > Math.abs(b.x) ? a : b,
    )
    const edge = layout.edges.find((e) => e.to === outermost.id)!

    // Reading the axis off the distance flips the outermost child, which is
    // further sideways than it is down.
    expect(Math.abs(outermost.x)).toBeGreaterThan(Math.abs(outermost.y))
    expect(edge.control1.x).toBe(edge.fromPoint.x)
    expect(edge.control1.y).toBeGreaterThan(edge.fromPoint.y)
    expect(edge.control2.x).toBe(edge.toPoint.x)
    expect(edge.control2.y).toBeLessThan(edge.toPoint.y)
  })

  it('runs sideways for a role that is seated sideways', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = focusOf(layout)
    const jump = withRole(layout, 'jump')[0]!
    const edge = layout.edges.find((e) => e.from === jump.id)!

    expect(edge.fromPoint).toStrictEqual({ x: jump.x + jump.width / 2, y: jump.y })
    expect(edge.toPoint).toStrictEqual({ x: focus.x - focus.width / 2, y: focus.y })
    expect(edge.control1.y).toBe(edge.fromPoint.y)
    expect(edge.control2.y).toBe(edge.toPoint.y)
  })

  it('starts and ends on a border, not in a centre', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const byId = new Map(layout.nodes.map((node) => [node.id, node]))

    for (const edge of layout.edges) {
      const from = byId.get(edge.from)!
      const to = byId.get(edge.to)!
      // On the border means: on one pair of sides, and within the other.
      const onBorder = (point: { x: number; y: number }, node: PlacedNode) => {
        const dx = Math.abs(point.x - node.x)
        const dy = Math.abs(point.y - node.y)
        const touchesSide = Math.abs(dx - node.width / 2) < 1e-9
        const touchesTop = Math.abs(dy - node.height / 2) < 1e-9
        return (
          (touchesSide || touchesTop) &&
          dx <= node.width / 2 + 1e-9 &&
          dy <= node.height / 2 + 1e-9
        )
      }
      expect(onBorder(edge.fromPoint, from), `from ${edge.from}`).toBe(true)
      expect(onBorder(edge.toPoint, to), `to ${edge.to}`).toBe(true)
    }
  })

  it('joins two nodes on the same line side to side, not under and back', () => {
    // The role says vertical, but there is no vertical room between them.
    const layout = arrangePlex(neighbourhoods.diamond)
    const edge = layout.edges.find(
      (e) => e.from === 'parent-0' && e.to === 'parent-1',
    )
    expect(edge).toBeDefined()
    expect(edge!.label).toBe('contains')

    const left = layout.nodes.find((n) => n.id === 'parent-0')!
    const right = layout.nodes.find((n) => n.id === 'parent-1')!
    expect(edge!.fromPoint).toStrictEqual({ x: left.x + left.width / 2, y: left.y })
    expect(edge!.toPoint).toStrictEqual({ x: right.x - right.width / 2, y: right.y })
    expect(edge!.control1.x).toBeGreaterThan(edge!.fromPoint.x)
    expect(edge!.control2.x).toBeLessThan(edge!.toPoint.x)
  })

  it('drops an edge from a node to itself', () => {
    const nodes = [{ id: 'a', label: 'A', role: 'focus' as const }]
    const layout = arrangePlex({ nodes, edges: [{ from: 'a', to: 'a' }] })
    expect(layout.edges).toHaveLength(0)
  })
})

