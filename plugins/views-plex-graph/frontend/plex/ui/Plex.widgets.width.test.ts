/** The widgets of a node wrap at the width the arrangement reserves for them, not at the width the arrangement gives the node afterwards. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import type { Size } from '../lib/arrange'
import type { Viewport } from '@numen/ui'
import type { PlexGraph } from '../../types'

const GRAPH: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    { id: 'parent-0', title: 'Hydrology', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'child' },
    { id: 'child-1', title: 'Spring', relation: 'child' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0' },
    { from: 'focus', to: 'child-1' },
  ],
}

const createViewport = (size: Size): Viewport => ({
  watch: (_of, took) => {
    took(size)
    return () => undefined
  },
  watchFreeSpace: (took) => {
    took(size)
    return () => undefined
  },
  watchWhole: () => () => undefined,
})

const names = defineComponent({
  props: { path: { type: String, required: true } },
  setup: (props) => () => h('span', { 'data-widget': 'names' }, props.path),
})

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

const getWidths = (plex: ReturnType<typeof mount>) =>
  plex.findAll('.plex__widgets').map((one) => Number(one.attributes('width')))

describe('the widths of the widgets of the nodes', () => {
  it('are one width for every node, the focus and the others, whatever width the arrangement gives each node', async () => {
    const plex = mount(Plex, {
      props: {
        graph: GRAPH,
        duration: 0,
        viewport: createViewport({ width: 1200, height: 900 }),
        getPath: (id: string) => `notes/${id}.md`,
        widgetsAbove: [{ key: 'org.example.names/view/names', component: names }],
      },
      attachTo: document.body,
    })
    await flushPromises()
    await flushPromises()

    const widths = getWidths(plex)
    expect(widths.length).toBe(GRAPH.nodes.length)
    expect(new Set(widths).size).toBe(1)
  })
})
