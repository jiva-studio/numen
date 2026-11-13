/** A note that is in both frames keeps the opacity it has when a movement is retargeted. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { interpolatePlex } from './interpolate'
import type { PlexFrame } from '../frame'
import type { PlexGraph } from '../../../types'

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'a', title: 'A', relation: 'child' },
    { id: 'b', title: 'B', relation: 'child' },
  ],
  edges: [],
}

const settled = arrangePlex(graph)
const fade = (frame: PlexFrame, id: string, opacity: number): PlexFrame => ({
  ...frame,
  nodes: frame.nodes.map((node) => (node.id === id ? { ...node, opacity } : node)),
})
const opacityOf = (frame: PlexFrame, id: string) =>
  frame.nodes.find((node) => node.id === id)!.opacity

describe('a movement that starts from a frame in the middle of another', () => {
  it('moves a half-faded note towards full opacity from where it is', () => {
    const middle = fade(settled, 'a', 0.4)

    expect(opacityOf(interpolatePlex(middle, settled, 0.5), 'a')).toBeCloseTo(0.7)
    expect(opacityOf(interpolatePlex(middle, settled, 0.01), 'a')).toBeLessThan(0.45)
  })

  it('does not bring an almost gone note back to full opacity at once', () => {
    const middle = fade(settled, 'b', 0.05)

    expect(opacityOf(interpolatePlex(middle, settled, 0.1), 'b')).toBeLessThan(0.2)
  })

  it('keeps a settled note opaque', () => {
    expect(opacityOf(interpolatePlex(settled, arrangePlex(graph), 0.5), 'a')).toBe(1)
  })
})
