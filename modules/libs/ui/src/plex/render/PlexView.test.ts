/** The renderer, given a frame directly — including one mid-movement. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, interpolatePlex } from '../arrange'
import type { PlexNeighbourhood } from '../model'

const before: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', label: 'Start', role: 'focus' },
    { id: 'staying', label: 'Staying', role: 'child' },
    { id: 'going', label: 'Going', role: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'focus', to: 'going' },
  ],
}

const after: PlexNeighbourhood = {
  nodes: [
    { id: 'staying', label: 'Staying', role: 'focus' },
    { id: 'focus', label: 'Start', role: 'parent' },
    { id: 'arriving', label: 'Arriving', role: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'staying', to: 'arriving' },
  ],
}

/** A third of the way across: one node leaving, one arriving, one travelling. */
const midMove = interpolatePlex(arrangePlex(before), arrangePlex(after), 0.3)

const VIEWPORT = { width: 1200, height: 800 }

const mountView = () =>
  mount(PlexView, { props: { frame: midMove, viewport: VIEWPORT } })

describe('a node that is not fully there', () => {
  it('cannot be chosen by clicking it', async () => {
    const view = mountView()
    await view.get('[aria-label^="Going"]').trigger('click')
    await view.get('[aria-label^="Arriving"]').trigger('click')
    expect(view.emitted('activate')).toBeUndefined()
  })

  it('cannot be chosen from the keyboard either', async () => {
    const view = mountView()
    await view.get('[aria-label^="Going"]').trigger('keydown', { key: 'Enter' })
    expect(view.emitted('activate')).toBeUndefined()
  })

  it('is not stopped at by tab, and is not announced', () => {
    const going = mountView().get('[aria-label^="Going"]')
    expect(going.attributes('tabindex')).toBe('-1')
    expect(going.attributes('aria-hidden')).toBe('true')
  })
})

describe('a node that is fully there', () => {
  it('can still be chosen while the rest of the picture is moving', async () => {
    // Only what is fading is out of bounds, not everything during a move.
    const view = mountView()
    await view.get('[aria-label^="Start"]').trigger('click')
    expect(view.emitted('activate')).toStrictEqual([['focus']])
  })
})

describe('the window onto the drawing', () => {
  const boxOf = (frame = arrangePlex(before)) => {
    const view = mount(PlexView, { props: { frame, viewport: VIEWPORT } })
    const [x, y, w, h] = view.get('svg').attributes('viewBox')!.split(' ').map(Number)
    return { x: x!, y: y!, w: w!, h: h! }
  }

  it('is centred on the origin, which is where the focus is', () => {
    const box = boxOf()
    expect(box.x).toBe(-box.w / 2)
    expect(box.y).toBe(-box.h / 2)
  })

  it('is the same window whatever the neighbourhood holds', () => {
    // Fitting the window to the drawing would shrink every box and letter as
    // neighbours are added.
    const small = boxOf(arrangePlex(before))
    const large = boxOf(arrangePlex({ ...before, nodes: [...before.nodes] }))
    const crowded = boxOf(
      arrangePlex({
        nodes: [
          { id: 'focus', label: 'Start', role: 'focus' },
          ...Array.from({ length: 40 }, (_, i) => ({
            id: `c${i}`,
            label: `Child ${i}`,
            role: 'child' as const,
          })),
        ],
        edges: [],
      }),
    )
    expect(large).toStrictEqual(small)
    expect(crowded).toStrictEqual(small)
  })

  it('draws a box at the size the arrangement asked for, in pixels', () => {
    const frame = arrangePlex(before)
    const view = mount(PlexView, { props: { frame, viewport: VIEWPORT } })
    const focus = frame.nodes.find((n) => n.role === 'focus')!
    const rect = view.get('[aria-label^="Start"] rect')
    expect(Number(rect.attributes('width'))).toBe(focus.width)
    expect(Number(rect.attributes('height'))).toBe(focus.height)
  })
})

describe('what the drawing does with an opacity', () => {
  it('draws a departing node fainter than one that is staying', () => {
    const view = mountView()
    const going = Number(view.get('[aria-label^="Going"]').attributes('opacity'))
    const staying = Number(view.get('[aria-label^="Staying"]').attributes('opacity'))
    expect(going).toBeGreaterThan(0)
    expect(going).toBeLessThan(1)
    expect(staying).toBe(1)
  })

  it('fades the line along with the node it belongs to', () => {
    const view = mountView()
    const faded = view
      .findAll('path.plex__edge')
      .map((path) => Number(path.attributes('opacity')))
    expect(faded.some((value) => value > 0 && value < 1)).toBe(true)
  })
})
