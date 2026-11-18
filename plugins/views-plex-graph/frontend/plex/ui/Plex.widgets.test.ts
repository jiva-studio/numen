/** Plugins place widgets above and below the nodes of the plex; it draws them with each node and keeps room for them. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import type { Size } from '../lib/arrange'
import type { Viewport } from '@numen/ui'
import type { PlexGraph } from '../../types'

const FAMILY: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    { id: 'parent-0', title: 'Hydrology', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'child' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0' },
  ],
}

const WIDGET_HEIGHT = 40

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

const createWidget = (name: string) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    render() {
      return h('span', { 'data-widget': name }, this.path)
    },
  })

const widget = (name: string) => ({
  key: `org.example.${name}/view/${name}`,
  component: createWidget(name),
})

let rings: (() => void)[] = []

beforeEach(() => {
  rings = []
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(private readonly ring: ResizeObserverCallback) {}
      observe() {
        rings.push(() => this.ring([], this as unknown as ResizeObserver))
      }
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
  // A document without a layout engine: whatever holds a widget is as tall as the widget.
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const height =
      this.matches('[data-widget]') || this.querySelector('[data-widget]') ? WIDGET_HEIGHT : 0
    return {
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 100,
      bottom: height,
      width: 100,
      height,
    } as DOMRect
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

const mountPlex = async (props: Partial<InstanceType<typeof Plex>['$props']> = {}) => {
  const plex = mount(Plex, {
    props: {
      graph: FAMILY,
      duration: 0,
      viewport: createViewport({ width: 1200, height: 1200 }),
      ...props,
    },
    attachTo: document.body,
  })
  await flushPromises()
  for (const ring of rings) ring()
  await flushPromises()
  return plex
}

const getPathOf = (id: string) => `notes/${id}.md`

const getY = (plex: Awaited<ReturnType<typeof mountPlex>>, title: string): number => {
  const at = plex.get(`[aria-label^="${title}"]`).attributes('transform') ?? ''
  return Number(/translate\((-?[\d.]+) (-?[\d.]+)\)/.exec(at)?.[2])
}

const readWidgets = (plex: Awaited<ReturnType<typeof mountPlex>>, name: string, path: string) =>
  plex.findAll(`[data-widget="${name}"]`).filter((one) => one.text() === path)

describe('the widgets of the plex', () => {
  it('are drawn above and below every node, the focus included, each given the path the caller names for the node', async () => {
    const plex = await mountPlex({
      widgetsAbove: [widget('parents')],
      widgetsBelow: [widget('tags')],
      getPath: getPathOf,
    })

    for (const node of FAMILY.nodes) {
      expect(readWidgets(plex, 'parents', getPathOf(node.id))).toHaveLength(1)
      expect(readWidgets(plex, 'tags', getPathOf(node.id))).toHaveLength(1)
    }
  })

  it('are drawn in the order of the entries on each side of a node', async () => {
    const plex = await mountPlex({
      widgetsAbove: [widget('parents'), widget('tags')],
      widgetsBelow: [widget('related'), widget('children')],
      getPath: getPathOf,
    })

    const forNode = plex
      .findAll('[data-widget]')
      .filter((one) => one.text() === getPathOf('child-0'))
      .map((one) => one.attributes('data-widget'))

    expect(forNode).toStrictEqual(['parents', 'tags', 'related', 'children'])
  })

  it('are drawn with no more than the notes the plex shows', async () => {
    const plex = await mountPlex({ widgetsAbove: [widget('tags')], getPath: getPathOf })

    expect(plex.findAll('[data-widget="tags"]')).toHaveLength(FAMILY.nodes.length)
  })

  it('are not drawn where no plugin places any', async () => {
    const plex = await mountPlex({ getPath: getPathOf })

    expect(plex.find('[data-widget]').exists()).toBe(false)
  })

  it('keep the nodes they sit between farther apart by the height they take', async () => {
    const without = await mountPlex()
    const gapWithout = getY(without, 'Watershed') - getY(without, 'Hydrology')
    without.unmount()

    const plex = await mountPlex({ widgetsBelow: [widget('tags')], getPath: getPathOf })
    const gapWith = getY(plex, 'Watershed') - getY(plex, 'Hydrology')

    expect(gapWith).toBeGreaterThanOrEqual(gapWithout + WIDGET_HEIGHT)
  })

  it('keep the nodes apart over a node as well', async () => {
    const without = await mountPlex()
    const gapWithout = getY(without, 'River delta') - getY(without, 'Watershed')
    without.unmount()

    const plex = await mountPlex({ widgetsAbove: [widget('parents')], getPath: getPathOf })
    const gapWith = getY(plex, 'River delta') - getY(plex, 'Watershed')

    expect(gapWith).toBeGreaterThanOrEqual(gapWithout + WIDGET_HEIGHT)
  })
})

describe('the widgets of the focus', () => {
  it('keep its children apart by the height they take under it', async () => {
    const without = await mountPlex()
    const gapWithout = getY(without, 'River delta') - getY(without, 'Watershed')
    without.unmount()

    const plex = await mountPlex({ widgetsBelow: [widget('tags')], getPath: getPathOf })
    const gapWith = getY(plex, 'River delta') - getY(plex, 'Watershed')

    expect(gapWith).toBeGreaterThanOrEqual(gapWithout + WIDGET_HEIGHT)
  })

  it('keep its parents apart by the height they take over it', async () => {
    const without = await mountPlex()
    const gapWithout = getY(without, 'Watershed') - getY(without, 'Hydrology')
    without.unmount()

    const plex = await mountPlex({ widgetsAbove: [widget('parents')], getPath: getPathOf })
    const gapWith = getY(plex, 'Watershed') - getY(plex, 'Hydrology')

    expect(gapWith).toBeGreaterThanOrEqual(gapWithout + WIDGET_HEIGHT)
  })
})
