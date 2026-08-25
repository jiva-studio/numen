/** The renderer, given a frame directly — including one mid-movement. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS, interpolatePlex } from '../arrange'
import type { PlexNeighbourhood } from '../model'

const before: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', title: 'Start', seat: 'focus' },
    { id: 'staying', title: 'Staying', seat: 'child' },
    { id: 'going', title: 'Going', seat: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'focus', to: 'going' },
  ],
}

const after: PlexNeighbourhood = {
  nodes: [
    { id: 'staying', title: 'Staying', seat: 'focus' },
    { id: 'focus', title: 'Start', seat: 'parent' },
    { id: 'arriving', title: 'Arriving', seat: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'staying', to: 'arriving' },
  ],
}

/** One title down the page and one across it, the second longer than its gap. */
const titled: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', title: 'Start', seat: 'focus' },
    { id: 'below', title: 'Below', seat: 'child' },
    { id: 'aside', title: 'Aside', seat: 'jump' },
  ],
  edges: [
    { from: 'focus', to: 'below', label: 'holds' },
    { from: 'focus', to: 'aside', label: 'the scene in the assembly' },
  ],
}

/** A third of the way across: one node leaving, one arriving, one travelling. */
const midMove = interpolatePlex(arrangePlex(before), arrangePlex(after), 0.3)

const VIEWPORT = { width: 1200, height: 800 }

const mountView = () =>
  mount(PlexView, {
    props: {
      frame: midMove,
      viewport: VIEWPORT,
      nodeSize: DEFAULT_OPTIONS.nodeSize,
    },
  })

/**
 * What a node decides for itself is tested where it lives, next to the node.
 * What is left here is the translation: a node says only that it was chosen,
 * and the view says which node that was.
 */
describe('what a node says, and who it was', () => {
  it('names the node that was chosen', async () => {
    // Also that only what is fading is out of bounds, not everything mid-move.
    const view = mountView()
    await view.get('[aria-label^="Start"]').trigger('click')
    expect(view.emitted('activate')).toStrictEqual([['focus']])
  })
})

describe('what each node is to a gesture', () => {
  const mountDuring = (props: Record<string, unknown>) =>
    mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        ...props,
      },
    })

  it('offers a handle from the node it left from, and from no other', async () => {
    const view = mountDuring({ gestureFrom: 'staying' })
    // A hand that has moved on and come to rest over some other node must not
    // be offered a second gesture while the first is still under way.
    await view.get('[aria-label^="Going"]').trigger('pointerenter')
    expect(view.findAll('.plex__handle')).toHaveLength(1)
    expect(view.get('[aria-label^="Staying"]').find('.plex__handle').exists()).toBe(true)
  })

  it('marks the node a link would be made to, and only that one', () => {
    const view = mountDuring({
      gestureFrom: 'focus',
      gestureAt: { x: 0, y: 200 },
      gestureOutcome: { kind: 'link', from: 'focus', to: 'going', seat: 'child' },
    })
    expect(view.findAll('.plex__node--target')).toHaveLength(1)
    expect(view.get('[aria-label^="Going"]').classes()).toContain('plex__node--target')
  })

  it('draws the node it would make where the pointer is, saying which seat', () => {
    const view = mountDuring({
      gestureFrom: 'focus',
      gestureAt: { x: 40, y: -200 },
      gestureOutcome: { kind: 'create', from: 'focus', seat: 'parent' },
    })
    const ghost = view.get('.plex__node--ghost')
    expect(ghost.attributes('transform')).toBe('translate(40 -200)')
    expect(ghost.get('.plex__title-text').text()).toBe('parent')
    expect(Number(ghost.get('rect').attributes('width'))).toBe(DEFAULT_OPTIONS.nodeSize.width)
  })
})

describe('the window onto the drawing', () => {
  const boxOf = (frame = arrangePlex(before)) => {
    const view = mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
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
          { id: 'focus', title: 'Start', seat: 'focus' },
          ...Array.from({ length: 40 }, (_, i) => ({
            id: `c${i}`,
            title: `Child ${i}`,
            seat: 'child' as const,
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
    const view = mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
    const focus = frame.nodes.find((n) => n.seat === 'focus')!
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

/**
 * A title runs into the box beside it, and the boxes are painted after the
 * titles. What answers that is the line the hand is on, drawn again after them.
 */
describe('the line under the hand', () => {
  const titledFrame = arrangePlex(titled)
  const down = titledFrame.edges.findIndex((edge) => edge.heading === 'none')
  const across = titledFrame.edges.findIndex((edge) => edge.heading === 'left')

  const mountTitled = () =>
    mount(PlexView, {
      props: {
        frame: titledFrame,
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })

  /** The band drawn over one line, in the order the frame gave the edges. */
  const reach = (view: ReturnType<typeof mountTitled>, at: number) =>
    view.findAll('.plex__edge-hit')[at]!

  it('lifts the line the hand comes to rest on, and puts it back after', async () => {
    const view = mountTitled()
    expect(view.find('.plex__lift').exists()).toBe(false)

    await reach(view, across).trigger('pointerenter')
    expect(view.find('.plex__lift').exists()).toBe(true)

    await reach(view, across).trigger('pointerleave')
    expect(view.find('.plex__lift').exists()).toBe(false)
  })

  it('puts the lifted line down again as soon as the picture moves', async () => {
    const view = mountTitled()
    await reach(view, across).trigger('pointerenter')
    expect(view.find('.plex__lift').exists()).toBe(true)

    // A move hands the view a frame a frame at a time, and the hand is left
    // pointing at wherever the line used to be.
    await view.setProps({ frame: arrangePlex(titled) })

    expect(view.find('.plex__lift').exists()).toBe(false)
  })

  it('draws the lifted line after every box, so it stands over them', async () => {
    const view = mountTitled()
    await reach(view, across).trigger('pointerenter')
    const lift = view.get('.plex__lift').element

    for (const node of view.findAll('.plex__node')) {
      const where = node.element.compareDocumentPosition(lift)
      expect(where & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
  })

  it('draws the lifted line once, and not also where it was resting', async () => {
    const view = mountTitled()
    const lines = titledFrame.edges.length

    expect(view.findAll('.plex__edge')).toHaveLength(lines)
    expect(view.findAll('.plex__edge-label')).toHaveLength(lines)

    await reach(view, across).trigger('pointerenter')

    expect(view.findAll('.plex__edge')).toHaveLength(lines)
    expect(view.get('.plex__lift .plex__edge').attributes('d')).toBe(
      reach(view, across).attributes('d'),
    )

    // Its title leaves the resting layer for the lift, where it is painted
    // twice over.
    expect(view.get('.plex__lift').findAll('.plex__edge-label')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(lines + 1)
  })

  it('paints the lifted title as a halo under its letters, on one line', async () => {
    const view = mountTitled()
    await reach(view, across).trigger('pointerenter')
    const [halo, letters] = view.get('.plex__lift').findAll('.plex__edge-label')

    // A glyph set along a path is painted as a run of its own, so a halo drawn
    // with the letters lies over the one beside it.
    expect(halo!.classes()).toContain('plex__edge-label--halo')
    expect(halo!.attributes('fill')).toBe('none')
    expect(letters!.classes()).toContain('plex__edge-label--letters')
    expect(letters!.attributes('stroke')).toBe('none')

    // One lands on the other: the same words, on the same line, at the same
    // place along it.
    expect(halo!.text()).toBe(letters!.text())
    for (const named of ['href', 'startOffset']) {
      expect(halo!.get('textPath').attributes(named)).toBe(
        letters!.get('textPath').attributes(named),
      )
      expect(halo!.get('textPath').attributes(named)).toBeDefined()
    }
  })

  it('paints a flat lifted title in the same two layers', async () => {
    const view = mountTitled()
    await reach(view, down).trigger('pointerenter')
    const titles = view.get('.plex__lift').findAll('.plex__edge-label')

    expect(titles).toHaveLength(2)
    expect(titles[0]!.attributes('fill')).toBe('none')
    expect(titles[1]!.attributes('stroke')).toBe('none')
    expect(titles[0]!.attributes('x')).toBe(titles[1]!.attributes('x'))
    expect(titles[0]!.find('textPath').exists()).toBe(false)
  })

  it('sets a title along its line where the line runs across the page', () => {
    const view = mountTitled()
    const titles = view.findAll('.plex__edge-label')

    expect(titles[across]!.find('textPath').exists()).toBe(true)
    // A line holding no one direction takes flat words at the midpoint.
    expect(titles[down]!.find('textPath').exists()).toBe(false)
    expect(titles[down]!.attributes('x')).toBeDefined()
  })

  it('sets a title on a line running the way the words are read', () => {
    const view = mountTitled()
    const edge = titledFrame.edges[across]!
    expect(edge.heading).toBe('left')

    const href = view.get('textPath').attributes('href')!
    const along = view.get(`defs path[id="${href.slice(1)}"]`)
    const numbers = along.attributes('d')!.match(/-?\d+(?:\.\d+)?/g)!.map(Number)

    expect(numbers[0]).toBe(edge.toPoint.x)
    expect(numbers.at(-2)).toBe(edge.fromPoint.x)
  })
})

describe('a node keeps its own drawing across a change', () => {
  it('is the same element after the ones around it move', async () => {
    // A node holds state of its own — whether a hand is over it, whether the
    // keyboard is on it — and the browser holds focus on an element. Drawn by
    // position rather than by name, one node inherits another's element, and
    // both follow whatever happens to be in that place.
    const view = mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })
    const was = view.get('[aria-label^="Going"]').element

    // Departing nodes are appended after arriving ones, so positions shift.
    await view.setProps({ frame: midMove })

    expect(view.get('[aria-label^="Going"]').element).toBe(was)
  })
})
