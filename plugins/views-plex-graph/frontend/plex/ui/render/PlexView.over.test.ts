/**
 * The views of the slot across a node are drawn over the whole box of the node, in a layer above the notes, and
 * are given the path of the note and the size, position and scale of the node as it moves. What a pointer does on
 * them is what it does on the node, unless a view stops it.
 */
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, h, markRaw, nextTick, type Component } from 'vue'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS, interpolatePlex } from '../../lib/arrange'
import type { PlexGraph } from '../../../types'

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'below', title: 'Below', relation: 'child' },
  ],
  edges: [{ from: 'focus', to: 'below' }],
}

const later: PlexGraph = {
  nodes: [
    { id: 'below', title: 'Below', relation: 'focus' },
    { id: 'focus', title: 'Start', relation: 'parent' },
  ],
  edges: [{ from: 'focus', to: 'below' }],
}

const VIEWPORT = { width: 1200, height: 800 }
const PATHS: Record<string, string> = { focus: 'Start.md', below: 'Below.md' }

afterEach(() => {
  document.body.replaceChildren()
})

const createView = (stopsClicks = false): Component =>
  markRaw(
    defineComponent({
      name: 'OverView',
      props: {
        path: { type: String, required: true },
        x: { type: Number, required: true },
        y: { type: Number, required: true },
        width: { type: Number, required: true },
        height: { type: Number, required: true },
        scale: { type: Number, required: true },
      },
      render() {
        return h('div', {
          'data-over-view': this.path,
          onClick: stopsClicks ? (event: Event) => event.stopPropagation() : undefined,
        })
      },
    }),
  )

const mountView = async (overrides: Record<string, unknown> = {}) => {
  const view = mount(PlexView, {
    attachTo: document.body,
    props: {
      frame: arrangePlex(graph),
      viewport: VIEWPORT,
      nodeSize: DEFAULT_OPTIONS.nodeSize,
      widgetsOver: [{ key: 'org.example.over/view/over', component: createView() }],
      getPath: (id: string) => PATHS[id] ?? '',
      ...overrides,
    },
  })
  await nextTick()
  return view
}

const getViews = (view: VueWrapper) => view.findAll('[data-over-view]')

const getProps = (view: VueWrapper, path: string) =>
  view
    .findAllComponents({ name: 'OverView' })
    .find((one) => one.props('path') === path)!
    .props() as Record<string, unknown>

describe('the views across a node', () => {
  it('are drawn once for each note, each given the path of its note', async () => {
    const view = await mountView()

    expect(getViews(view).map((one) => one.attributes('data-over-view'))).toStrictEqual(
      expect.arrayContaining(['Start.md', 'Below.md']),
    )
    expect(getViews(view)).toHaveLength(2)
  })

  it('are drawn in a layer after the one open notes are drawn in, which comes after every note', async () => {
    const view = await mountView()
    const layer = view.get('.plex__over-layer').element
    const lift = view.get('.plex__lift').element
    const last = view.findAll('.plex__stack').at(-1)!.element

    expect(layer.contains(getViews(view)[0]!.element)).toBe(true)
    expect(lift.compareDocumentPosition(layer) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
    expect(last.compareDocumentPosition(layer) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
  })

  it('are given the middle, the size and the scale of the node, and the surface, in the units of the surface', async () => {
    const frame = arrangePlex(graph)
    const view = await mountView({ frame, nodeSize: DEFAULT_OPTIONS.nodeSize })
    const focus = frame.nodes.find((one) => one.id === 'focus')!
    const below = frame.nodes.find((one) => one.id === 'below')!

    expect(getProps(view, 'Start.md')).toMatchObject({
      path: 'Start.md',
      x: focus.x,
      y: focus.y,
      width: focus.width,
      height: focus.height,
      scale: focus.height / DEFAULT_OPTIONS.nodeSize.height,
      viewport: VIEWPORT,
    })
    expect(getProps(view, 'Below.md')).toMatchObject({
      x: below.x,
      y: below.y,
      width: below.width,
      height: below.height,
      scale: 1,
    })
  })

  it('are given a scale of at least one', async () => {
    const view = await mountView({ nodeSize: { width: 1000, height: 1000 } })

    expect(getProps(view, 'Below.md').scale).toBe(1)
  })

  it('follow the node while it moves', async () => {
    const from = arrangePlex(graph)
    const to = arrangePlex(later)
    const view = await mountView({ frame: from })
    const before = getProps(view, 'Below.md')
    const start = { x: before.x, y: before.y, width: before.width, height: before.height }

    await view.setProps({ frame: interpolatePlex(from, to, 0.5) })
    const midway = getProps(view, 'Below.md')
    const midwayNode = interpolatePlex(from, to, 0.5).nodes.find((one) => one.id === 'below')!

    expect(midway).toMatchObject({
      x: midwayNode.x,
      y: midwayNode.y,
      width: midwayNode.width,
      height: midwayNode.height,
    })
    expect([midway.x, midway.y, midway.width, midway.height]).not.toStrictEqual(
      Object.values(start),
    )
  })

  it('are not drawn where no plugin places one', async () => {
    const view = await mountView({ widgetsOver: [] })

    expect(getViews(view)).toHaveLength(0)
    expect(view.find('.plex__over-node').exists()).toBe(false)
  })
})

describe('a pointer on a view across a node', () => {
  it('is a pointer on the node: the node is marked while it is over the view', async () => {
    const view = await mountView()
    const node = view.get('[aria-label^="Below"]')

    const over = view.get('.plex__over-node[data-node="below"]')

    await over.trigger('pointerenter')
    expect(node.classes()).toContain('plex__node--pointer')

    await over.trigger('pointerleave')
    expect(node.classes()).not.toContain('plex__node--pointer')
  })

  it('is a pointer on the node: a click on the view chooses the note', async () => {
    const view = await mountView()

    await view.get('[data-over-view="Below.md"]').trigger('click')

    expect(view.emitted('activate')).toStrictEqual([['below']])
  })

  it('does not choose the note when the view does not let the click through', async () => {
    const view = await mountView({
      widgetsOver: [{ key: 'org.example.over/view/over', component: createView(true) }],
    })

    await view.get('[data-over-view="Below.md"]').trigger('click')

    expect(view.emitted('activate')).toBeUndefined()
  })
})
