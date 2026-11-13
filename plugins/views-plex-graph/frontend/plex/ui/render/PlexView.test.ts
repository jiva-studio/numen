import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexView from './PlexView.vue'
import PlexNodeView from './PlexNodeView.vue'
import { arrangePlex, DEFAULT_OPTIONS, interpolatePlex } from '../../lib/arrange'
import { getNamesRoom } from '../../lib/names'
import type { PlacedNode } from '../../lib/node'
import type { PlexGraph } from '../../../types'

const before: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'staying', title: 'Staying', relation: 'child' },
    { id: 'going', title: 'Going', relation: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'focus', to: 'going' },
  ],
}

const after: PlexGraph = {
  nodes: [
    { id: 'staying', title: 'Staying', relation: 'focus' },
    { id: 'focus', title: 'Start', relation: 'parent' },
    { id: 'arriving', title: 'Arriving', relation: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'staying' },
    { from: 'staying', to: 'arriving' },
  ],
}

const titled: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'below', title: 'Below', relation: 'child' },
    { id: 'aside', title: 'Aside', relation: 'jump' },
  ],
  edges: [
    { from: 'focus', to: 'below', labels: [{ text: 'holds' }] },
    { from: 'focus', to: 'aside', labels: [{ text: 'the scene in the assembly' }] },
  ],
}

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

describe('the order things are drawn in', () => {
  it('draws every node, and so the headings that open under it, over the names of all of them', () => {
    const room = getNamesRoom(
      { above: [{ id: 'a', title: 'Heat' }], below: [{ id: 'b', title: 'Work' }] },
      { fontSize: 10, width: 144, measure: (title) => title.length * 6 },
    )
    const view = mount(PlexView, {
      attachTo: document.body,
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        names: () => room,
      },
    })

    const lastNames = view.findAll('.plex__names').at(-1)!.element
    const firstNode = view.get('.plex__node').element
    expect(view.findAll('.plex__names').length).toBeGreaterThan(0)
    expect(
      lastNames.compareDocumentPosition(firstNode) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })
})

describe('node activation', () => {
  it('emits activate with the clicked node id', async () => {
    const view = mountView()
    await view.get('[aria-label^="Start"]').trigger('click')
    expect(view.emitted('activate')).toStrictEqual([['focus']])
  })
})

describe('gesture roles and ghost', () => {
  const mountPlexViewDuring = (props: Record<string, unknown>) =>
    mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        ...props,
      },
    })

  it('shows the handle only on the source node during a gesture', async () => {
    const view = mountPlexViewDuring({ gestureFrom: 'staying' })
    await view.get('[aria-label^="Going"]').trigger('pointerenter')
    expect(view.findAll('.plex__handle')).toHaveLength(1)
    expect(view.get('[aria-label^="Staying"]').find('.plex__handle').exists()).toBe(true)
  })

  it('marks only the link target node', () => {
    const view = mountPlexViewDuring({
      gestureFrom: 'focus',
      gestureAt: { x: 0, y: 200 },
      gestureOutcome: { kind: 'link', from: 'focus', to: 'going', relation: 'child' },
    })
    expect(view.findAll('.plex__node--target')).toHaveLength(1)
    expect(view.get('[aria-label^="Going"]').classes()).toContain('plex__node--target')
  })

  it('renders the ghost node at the pointer position with the relation name', () => {
    const view = mountPlexViewDuring({
      gestureFrom: 'focus',
      gestureAt: { x: 40, y: -200 },
      gestureOutcome: { kind: 'create', from: 'focus', relation: 'parent' },
    })
    const ghost = view.get('.plex__node--ghost')
    expect(ghost.attributes('transform')).toBe('translate(40 -200)')
    expect(ghost.get('.plex__title-text').text()).toBe('parent')
    expect(Number(ghost.get('rect').attributes('width'))).toBe(224)
  })

  it('draws the ghost at the size of a field, whatever size a note has', () => {
    const field = { width: 224, height: 32 }
    const sizes = [DEFAULT_OPTIONS.nodeSize, { width: 300, height: 80 }].map((nodeSize) => {
      const view = mountPlexViewDuring({
        nodeSize,
        ghostSize: field,
        gestureFrom: 'focus',
        gestureAt: { x: 40, y: -200 },
        gestureOutcome: { kind: 'create', from: 'focus', relation: 'parent' },
      })
      const rect = view.get('.plex__node--ghost rect')
      return [Number(rect.attributes('width')), Number(rect.attributes('height'))]
    })

    expect(sizes).toStrictEqual([
      [224, 32],
      [224, 32],
    ])
  })

  it('draws the ghost title at its size, however tall the ghost is', () => {
    const view = mountPlexViewDuring({
      ghostSize: { width: 224, height: 64 },
      gestureFrom: 'focus',
      gestureAt: { x: 40, y: -200 },
      gestureOutcome: { kind: 'create', from: 'focus', relation: 'parent' },
    })
    expect(view.get('.plex__node--ghost foreignObject').attributes('transform')).toBeUndefined()
  })
})

describe('viewport viewBox', () => {
  const getBox = (frame = arrangePlex(before)) => {
    const view = mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
    const [x, y, w, h] = view.get('svg').attributes('viewBox')!.split(' ').map(Number)
    return { x: x!, y: y!, w: w!, h: h! }
  }

  it('centers the viewBox on the origin', () => {
    const box = getBox()
    expect(box.x).toBe(-box.w / 2)
    expect(box.y).toBe(-box.h / 2)
  })

  it('keeps the same viewBox size regardless of node count', () => {
    const small = getBox(arrangePlex(before))
    const large = getBox(arrangePlex({ ...before, nodes: [...before.nodes] }))
    const crowded = getBox(
      arrangePlex({
        nodes: [
          { id: 'focus', title: 'Start', relation: 'focus' },
          ...Array.from({ length: 40 }, (_, i) => ({
            id: `c${i}`,
            title: `Child ${i}`,
            relation: 'child' as const,
          })),
        ],
        edges: [],
      }),
    )
    expect(large).toStrictEqual(small)
    expect(crowded).toStrictEqual(small)
  })

  it('renders node rects with the arranged pixel dimensions', () => {
    const frame = arrangePlex(before)
    const view = mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
    const focus = frame.nodes.find((n) => n.relation === 'focus')!
    const rect = view.get('[aria-label^="Start"] rect')
    expect(Number(rect.attributes('width'))).toBe(focus.width)
    expect(Number(rect.attributes('height'))).toBe(focus.height)
  })
})

describe('element opacity', () => {
  it('renders a departing node with reduced opacity', () => {
    const view = mountView()
    const going = Number(view.get('[aria-label^="Going"]').attributes('opacity'))
    const staying = Number(view.get('[aria-label^="Staying"]').attributes('opacity'))
    expect(going).toBeGreaterThan(0)
    expect(going).toBeLessThan(1)
    expect(staying).toBe(1)
  })

  it('renders a departing edge with reduced opacity', () => {
    const view = mountView()
    const faded = view.findAll('path.plex__edge').map((path) => Number(path.attributes('opacity')))
    expect(faded.some((value) => value > 0 && value < 1)).toBe(true)
  })
})

describe('hovered edge line', () => {
  const titledFrame = arrangePlex(titled)
  const down = titledFrame.edges.findIndex((edge) => edge.to === 'below')
  const acrossIndex = titledFrame.edges.findIndex((edge) => edge.to === 'aside')

  const mountTitledPlexView = () =>
    mount(PlexView, {
      props: {
        frame: titledFrame,
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })

  const reach = (view: ReturnType<typeof mountTitledPlexView>, pos: number) =>
    view.findAll('.plex__edge-hit')[pos]!

  const title = (view: ReturnType<typeof mountTitledPlexView>, pos: number) =>
    view.findAll('.plex__edge-label').slice(2 * pos, 2 * pos + 2)

  it('raises the hovered line on pointerenter and lowers it on pointerleave', async () => {
    const view = mountTitledPlexView()
    expect(view.find('.plex__raised').exists()).toBe(false)

    await reach(view, acrossIndex).trigger('pointerenter')
    expect(view.find('.plex__raised').exists()).toBe(true)

    await reach(view, acrossIndex).trigger('pointerleave')
    expect(view.find('.plex__raised').exists()).toBe(false)
  })

  it('lowers the raised line when the frame changes', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')
    expect(view.find('.plex__raised').exists()).toBe(true)

    await view.setProps({ frame: arrangePlex(titled) })

    expect(view.find('.plex__raised').exists()).toBe(false)
  })

  it('renders the raised line after all node elements', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')
    const raised = view.get('.plex__raised').element

    for (const node of view.findAll('.plex__node')) {
      const where = node.element.compareDocumentPosition(raised)
      expect(where & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
  })

  it('moves the hovered line and its title from the resting layer to the raised layer', async () => {
    const view = mountTitledPlexView()
    const lines = titledFrame.edges.length

    expect(view.findAll('.plex__edge')).toHaveLength(lines)
    expect(view.findAll('.plex__edge-label')).toHaveLength(2 * lines)

    await reach(view, acrossIndex).trigger('pointerenter')

    expect(view.findAll('.plex__edge')).toHaveLength(lines)
    expect(view.get('.plex__raised .plex__edge').attributes('d')).toBe(
      reach(view, acrossIndex).attributes('d'),
    )

    expect(view.get('.plex__raised').findAll('.plex__edge-label')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(2 * lines)
  })

  it('renders a resting title with halo and letter layers on the same path', () => {
    const view = mountTitledPlexView()
    const [halo, letters] = title(view, acrossIndex)

    expect(halo!.classes()).toContain('plex__edge-label--halo')
    expect(letters!.classes()).toContain('plex__edge-label--letters')

    expect(halo!.text()).toBe(letters!.text())
    expect(halo!.get('textPath').attributes('href')).toBe(
      letters!.get('textPath').attributes('href'),
    )
  })

  it('renders a raised title with halo and letter layers on the same path', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')
    const [halo, letters] = view.get('.plex__raised').findAll('.plex__edge-label')

    expect(halo!.classes()).toContain('plex__edge-label--halo')
    expect(letters!.classes()).toContain('plex__edge-label--letters')

    expect(halo!.text()).toBe(letters!.text())
    for (const named of ['href', 'startOffset']) {
      expect(halo!.get('textPath').attributes(named)).toBe(
        letters!.get('textPath').attributes(named),
      )
      expect(halo!.get('textPath').attributes(named)).toBeDefined()
    }
  })

  it('renders every edge title inside a textPath element', () => {
    const view = mountTitledPlexView()

    for (const targetIndex of [down, acrossIndex]) {
      expect(title(view, targetIndex)[0]!.find('textPath').exists()).toBe(true)
      expect(title(view, targetIndex)[0]!.attributes('x')).toBeUndefined()
    }
  })

  it('renders the truncated title text from the arrangement', () => {
    const arrangedPlex = arrangePlex(titled, { measureLabel: (label) => 30 * label.length })
    const view = mount(PlexView, {
      props: { frame: arrangedPlex, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
    const edge = arrangedPlex.edges[acrossIndex]!

    expect(edge.labels?.[0]?.text).toBe('the scene in the assembly')
    expect(edge.titles[0]!.words.endsWith('…')).toBe(true)
    expect(title(view, acrossIndex)[0]!.text()).toBe(edge.titles[0]?.words)
  })

  it('reverses the reading path when the title heading is against', () => {
    const view = mountTitledPlexView()
    const edge = titledFrame.edges[acrossIndex]!
    expect(edge.titles[0]!.heading).toBe('against')

    const href = title(view, acrossIndex)[0]!.get('textPath').attributes('href')!
    const line = view.get(`defs path[id="${href.slice(1)}"]`)
    const numbers = line
      .attributes('d')!
      .match(/-?\d+(?:\.\d+)?/g)!
      .map(Number)

    expect(numbers[0]).toBe(edge.toPoint.x)
    expect(numbers.at(-2)).toBe(edge.fromPoint.x)
  })
})

describe('bidirectional edge pair', () => {
  const paired = arrangePlex({
    nodes: [
      { id: 'focus', title: 'Start', relation: 'focus' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [
      { from: 'focus', to: 'aside', labels: [{ text: 'out' }] },
      { from: 'aside', to: 'focus', labels: [{ text: 'back' }] },
    ],
  })

  it('raises both edges of a pair on pointerenter', async () => {
    const view = mount(PlexView, {
      props: { frame: paired, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })
    expect(view.findAll('.plex__edge')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(4)

    await view.findAll('.plex__edge-hit')[0]!.trigger('pointerenter')

    expect(view.findAll('.plex__edge')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(4)
    expect(view.get('.plex__raised').findAll('.plex__edge')).toHaveLength(2)
  })
})

describe('node element identity across frames', () => {
  it('preserves the DOM element of a node when other nodes move', async () => {
    const view = mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })
    const was = view.get('[aria-label^="Going"]').element

    await view.setProps({ frame: midMove })

    expect(view.get('[aria-label^="Going"]').element).toBe(was)
  })
})

describe('settled node draw order', () => {
  const getDrawnOrder = (view: ReturnType<typeof mountView>) =>
    view.findAll('.plex__node').map((node) => node.attributes('aria-label'))

  const getBox = (view: ReturnType<typeof mountView>, name: string) =>
    view.findAllComponents(PlexNodeView).find((node) => node.props('node').id === name)!

  const mountRestingPlexView = () =>
    mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })

  it('renders the settled node last so it covers neighbors', async () => {
    const view = mountRestingPlexView()
    expect(getDrawnOrder(view)).toStrictEqual(['Start, focus', 'Staying, child', 'Going, child'])

    getBox(view, 'staying').vm.$emit('settle', true)
    await view.vm.$nextTick()
    expect(getDrawnOrder(view)).toStrictEqual(['Start, focus', 'Going, child', 'Staying, child'])
  })

  it('preserves the DOM element when the node moves to the end', async () => {
    const view = mountRestingPlexView()
    const was = view.get('[aria-label^="Staying"]').element

    getBox(view, 'staying').vm.$emit('settle', true)
    await view.vm.$nextTick()

    expect(view.get('[aria-label^="Staying"]').element).toBe(was)
  })

  it('restores the original draw order when the node unsettles', async () => {
    const view = mountRestingPlexView()
    getBox(view, 'staying').vm.$emit('settle', true)
    await view.vm.$nextTick()

    getBox(view, 'staying').vm.$emit('settle', false)
    await view.vm.$nextTick()
    expect(getDrawnOrder(view)).toStrictEqual(['Start, focus', 'Staying, child', 'Going, child'])
  })

  it('passes the widened box to each node', () => {
    const wideBox = { width: 400, offset: -12 }
    const view = mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        widen: (node: PlacedNode) => (node.id === 'going' ? wideBox : null),
      },
    })

    expect(getBox(view, 'going').props('wide')).toStrictEqual(wideBox)
    expect(getBox(view, 'staying').props('wide')).toBeNull()
  })
})

describe('edge arrowheads', () => {
  const named: PlexGraph = {
    nodes: [
      { id: 'focus', title: 'Start', relation: 'focus' },
      { id: 'below', title: 'Below', relation: 'child' },
      { id: 'above', title: 'Above', relation: 'parent' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [
      { from: 'focus', to: 'below', labels: [{ text: 'holds' }], arrows: ['to'] },
      { from: 'above', to: 'focus', arrows: ['from'] },
      { from: 'aside', to: 'focus', labels: [{ text: 'see also' }] },
    ],
  }

  const frame = arrangePlex(named)

  const mountNamedPlexView = () =>
    mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })

  const parseTransform = (transform: string) => {
    const [x, y, angle] = transform.match(/-?\d+(?:\.\d+)?/g)!.map(Number)
    return { x: x!, y: y!, angle: angle! }
  }

  it('renders an arrowhead for each edge with arrows', () => {
    expect(mountNamedPlexView().findAll('.plex__edge-arrow')).toHaveLength(2)
  })

  it('positions and rotates each arrowhead at its edge endpoint', () => {
    const view = mountNamedPlexView()
    const renderedArrows = view.findAll('.plex__edge-arrow')

    for (const [at, edge] of [frame.edges[0]!, frame.edges[1]!].entries()) {
      const head = parseTransform(renderedArrows[at]!.attributes('transform')!)
      expect(head.x).toBeCloseTo(edge.arrowheads![0]!.at.x)
      expect(head.y).toBeCloseTo(edge.arrowheads![0]!.at.y)
      expect(head.angle).toBeCloseTo(edge.arrowheads![0]!.angle)
    }

    expect(parseTransform(renderedArrows[0]!.attributes('transform')!).angle).toBeCloseTo(90)
    expect(parseTransform(renderedArrows[1]!.attributes('transform')!).angle).toBeCloseTo(-90)
  })

  it('applies the edge opacity to its arrowhead', () => {
    const view = mount(PlexView, {
      props: {
        frame: { ...frame, edges: frame.edges.map((edge) => ({ ...edge, opacity: 0.4 })) },
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })
    for (const arrow of view.findAll('.plex__edge-arrow')) {
      expect(arrow.attributes('opacity')).toBe('0.4')
    }
  })

  it('renders arrowheads before edge titles', () => {
    const view = mountNamedPlexView()
    const arrow = view.findAll('.plex__edge-arrow')[0]!.element
    for (const title of view.findAll('.plex__edge-label')) {
      const where = arrow.compareDocumentPosition(title.element)
      expect(where & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
  })

  it('raises the arrowhead with its hovered edge line', async () => {
    const view = mountNamedPlexView()
    await view.findAll('.plex__edge-hit')[0]!.trigger('pointerenter')

    expect(view.get('.plex__raised').findAll('.plex__edge-arrow')).toHaveLength(1)
    expect(view.findAll('.plex__edge-arrow')).toHaveLength(2)
  })
})

describe('the gesture with Pill background off', () => {
  const mountBare = (props: Record<string, unknown>) =>
    mount(PlexView, {
      props: {
        frame: arrangePlex(before),
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        showPills: false,
        ...props,
      },
    })

  it('draws no pill on any node', () => {
    const view = mountBare({})
    expect(view.findAll('.plex__node')).toHaveLength(3)
    expect(view.findAll('.plex__box')).toHaveLength(0)
  })

  it('draws the ring of the note a link is dragged over', () => {
    const view = mountBare({
      gestureFrom: 'focus',
      gestureAt: { x: 0, y: 200 },
      gestureOutcome: { kind: 'link', from: 'focus', to: 'going', relation: 'child' },
    })
    const target = view.get('[aria-label^="Going"]')
    const other = view.get('[aria-label^="Staying"]')

    expect(target.classes()).toContain('plex__node--target')
    expect(target.findAll('rect').length).toBeGreaterThan(other.findAll('rect').length)
  })

  it('draws the dashed outline of the note about to be made', () => {
    const view = mountBare({
      gestureFrom: 'focus',
      gestureAt: { x: 40, y: -200 },
      gestureOutcome: { kind: 'create', from: 'focus', relation: 'parent' },
    })
    expect(view.get('.plex__node--ghost').find('.plex__box').exists()).toBe(true)
  })
})
