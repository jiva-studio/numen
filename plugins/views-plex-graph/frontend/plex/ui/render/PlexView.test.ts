import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, createPath, DEFAULT_OPTIONS, interpolatePlex } from '../../lib/arrange'
import type { Drop } from '../../lib/arrange'
import type { Position } from '../../lib/node'
import type { PlexGraph } from '../../../types'

const gestureOf = (state: { from?: string; at?: Position; outcome?: Drop }) => ({
  from: ref(state.from ?? null),
  at: ref(state.at ?? null),
  outcome: ref(state.outcome ?? null),
})

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
    const view = mountPlexViewDuring({ gesture: gestureOf({ from: 'staying' }) })
    await view.get('[aria-label^="Going"]').trigger('pointerenter')
    expect(view.findAll('.plex__handle')).toHaveLength(1)
    expect(view.get('[aria-label^="Staying"]').find('.plex__handle').exists()).toBe(true)
  })

  it('marks only the link target node', () => {
    const view = mountPlexViewDuring({
      gesture: gestureOf({
        from: 'focus',
        at: { x: 0, y: 200 },
        outcome: { kind: 'link', from: 'focus', to: 'going', relation: 'child' },
      }),
    })
    expect(view.findAll('.plex__node--target')).toHaveLength(1)
    expect(view.get('[aria-label^="Going"]').classes()).toContain('plex__node--target')
  })

  it('renders the ghost node at the pointer position with the relation name', () => {
    const view = mountPlexViewDuring({
      gesture: gestureOf({
        from: 'focus',
        at: { x: 40, y: -200 },
        outcome: { kind: 'create', from: 'focus', relation: 'parent' },
      }),
    })
    const ghost = view.get('.plex__node--ghost')
    expect(ghost.attributes('transform')).toBe('translate(40 -200)')
    expect(ghost.get('.plex__title-text').text()).toBe('parent')
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
})

describe('the links of a hovered note', () => {
  const frame = arrangePlex({
    nodes: [
      { id: 'focus', title: 'Start', relation: 'focus' },
      { id: 'below', title: 'Below', relation: 'child' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [
      { from: 'focus', to: 'below' },
      { from: 'focus', to: 'aside' },
      { from: 'below', to: 'aside' },
    ],
  })

  const createLinkedView = (props: Record<string, unknown> = {}) =>
    mount(PlexView, {
      attachTo: document.body,
      props: {
        frame,
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
        highlightLinks: true,
        ...props,
      },
    })

  const note = (view: ReturnType<typeof createLinkedView>, title: string) =>
    view.get(`[aria-label^="${title}"]`)

  const raised = (view: ReturnType<typeof createLinkedView>) =>
    view.findAll('.plex__raised .plex__edge').map((edge) => edge.attributes('d'))

  const pathOf = (from: string, to: string) =>
    createPath(frame.edges.find((edge) => edge.from === from && edge.to === to)!)

  it('lifts every edge that joins the note and no other while the pointer is over it', async () => {
    const view = createLinkedView()
    expect(raised(view)).toStrictEqual([])

    await note(view, 'Below').trigger('pointerenter')

    expect(raised(view).sort()).toStrictEqual(
      [pathOf('focus', 'below'), pathOf('below', 'aside')].sort(),
    )
  })

  it('draws the lifted edges under every node, so what opens under a note covers them', async () => {
    const view = createLinkedView()
    await note(view, 'Below').trigger('pointerenter')

    const stack = view.get('.plex__stack').element
    const lifted = view.get('.plex__raised').element

    expect(lifted.compareDocumentPosition(stack) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('lifts the edges of another note when the pointer moves to it', async () => {
    const view = createLinkedView()
    await note(view, 'Below').trigger('pointerenter')
    await note(view, 'Below').trigger('pointerleave')
    await note(view, 'Aside').trigger('pointerenter')

    expect(raised(view)).toHaveLength(2)
    expect(raised(view)).toContain(pathOf('focus', 'aside'))
    expect(raised(view)).not.toContain(pathOf('focus', 'below'))
  })

  it('lowers the edges when the pointer leaves the note', async () => {
    const view = createLinkedView()
    await note(view, 'Below').trigger('pointerenter')
    await note(view, 'Below').trigger('pointerleave')

    expect(raised(view)).toStrictEqual([])
  })

  it('lifts nothing while the setting is off', async () => {
    const view = createLinkedView({ highlightLinks: false })
    await note(view, 'Below').trigger('pointerenter')

    expect(raised(view)).toStrictEqual([])
  })

  it('lifts nothing while a connection is dragged out of a note', async () => {
    const view = createLinkedView({
      gesture: gestureOf({ from: 'focus', at: { x: 0, y: 0 } }),
    })
    await note(view, 'Below').trigger('pointerenter')

    expect(raised(view)).toStrictEqual([])
  })

  it('lowers the edges when the note is removed under the pointer', async () => {
    const view = createLinkedView()
    await note(view, 'Below').trigger('pointerenter')

    await view.setProps({ frame: arrangePlex(before) })

    expect(view.find('.plex__raised').exists()).toBe(false)
  })
})

describe('hovered edge line', () => {
  const titledFrame = arrangePlex(titled)
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

  it('names each line by the two notes it joins', () => {
    const view = mountTitledPlexView()

    expect(reach(view, acrossIndex).attributes('role')).toBe('button')
    expect(reach(view, acrossIndex).attributes('aria-label')).toBe('Line from Start to Aside')
  })

  it('raises the hovered line on pointerenter and lowers it on pointerleave', async () => {
    const view = mountTitledPlexView()
    expect(view.find('.plex__raised').exists()).toBe(false)

    await reach(view, acrossIndex).trigger('pointerenter')
    expect(view.find('.plex__raised').exists()).toBe(true)

    await reach(view, acrossIndex).trigger('pointerleave')
    expect(view.find('.plex__raised').exists()).toBe(false)
  })

  it('lowers the raised line when the frame no longer has the line', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')
    expect(view.find('.plex__raised').exists()).toBe(true)

    await view.setProps({ frame: arrangePlex({ nodes: titled.nodes, edges: [] }) })

    expect(view.find('.plex__raised').exists()).toBe(false)
  })

  it('keeps the raised line through a change of the frame that still has the line', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')

    await view.setProps({ frame: arrangePlex(titled) })

    expect(view.find('.plex__raised').exists()).toBe(true)
  })

  it('renders the raised line before all node elements', async () => {
    const view = mountTitledPlexView()
    await reach(view, acrossIndex).trigger('pointerenter')
    const raised = view.get('.plex__raised').element

    for (const node of view.findAll('.plex__node')) {
      const where = node.element.compareDocumentPosition(raised)
      expect(where & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy()
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

  it('renders the truncated title text from the arrangement', () => {
    const arrangedPlex = arrangePlex(titled, { measureLabel: (label) => 30 * label.length })
    const view = mount(PlexView, {
      props: {
        frame: arrangedPlex,
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
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
      props: {
        frame: paired,
        viewport: VIEWPORT,
        nodeSize: DEFAULT_OPTIONS.nodeSize,
      },
    })
    expect(view.findAll('.plex__edge')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(4)

    await view.findAll('.plex__edge-hit')[0]!.trigger('pointerenter')

    expect(view.findAll('.plex__edge')).toHaveLength(2)
    expect(view.findAll('.plex__edge-label')).toHaveLength(4)
    expect(view.get('.plex__raised').findAll('.plex__edge')).toHaveLength(2)
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

  const mountWithNamesPlexView = () =>
    mount(PlexView, {
      props: { frame, viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
    })

  const parseTransform = (transform: string) => {
    const [x, y, angle] = transform.match(/-?\d+(?:\.\d+)?/g)!.map(Number)
    return { x: x!, y: y!, angle: angle! }
  }

  it('renders an arrowhead for each edge with arrows', () => {
    expect(mountWithNamesPlexView().findAll('.plex__edge-arrow')).toHaveLength(2)
  })

  it('positions and rotates each arrowhead at its edge endpoint', () => {
    const view = mountWithNamesPlexView()
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

  it('raises the arrowhead with its hovered edge line', async () => {
    const view = mountWithNamesPlexView()
    await view.findAll('.plex__edge-hit')[0]!.trigger('pointerenter')

    expect(view.get('.plex__raised').findAll('.plex__edge-arrow')).toHaveLength(1)
    expect(view.findAll('.plex__edge-arrow')).toHaveLength(2)
  })
})
