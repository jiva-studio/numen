/** A side whose widgets draw nothing keeps no room; it gets its room when a widget starts drawing, and gives it back when the last one stops. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref, type Ref } from 'vue'
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
const NODE_GAP = 8

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

/** A widget that draws a span while `drawing` is on and nothing at all while it is off, like the tags of a note without tags. */
const createWidget = (name: string, drawing: Ref<boolean>) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    render() {
      return drawing.value ? h('span', { 'data-widget': name }, this.path) : null
    },
  })

const createEntry = (name: string, drawing: Ref<boolean>) => ({
  key: `org.example.${name}/view/${name}`,
  component: createWidget(name, drawing),
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
  // A document without a layout engine: a widget is as tall as WIDGET_HEIGHT, and the group of widgets
  // is as tall as its widgets and the node gap it carries as padding.
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const drawn = this.querySelectorAll('[data-widget]').length + (this.matches('[data-widget]') ? 1 : 0)
    const height = this.matches('.plex__widgets-group')
      ? NODE_GAP + drawn * WIDGET_HEIGHT
      : drawn > 0
        ? WIDGET_HEIGHT
        : 0
    return { x: 0, y: 0, top: 0, left: 0, right: 100, bottom: height, width: 100, height } as DOMRect
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

type Mounted = Awaited<ReturnType<typeof mountPlex>>

const settle = async () => {
  await flushPromises()
  for (const ring of rings) ring()
  await flushPromises()
}

const mountPlex = async (props: Partial<InstanceType<typeof Plex>['$props']> = {}) => {
  const plex = mount(Plex, {
    props: {
      graph: FAMILY,
      duration: 0,
      viewport: createViewport({ width: 1200, height: 1200 }),
      getPath: (id: string) => `notes/${id}.md`,
      ...props,
    },
    attachTo: document.body,
  })
  await settle()
  return plex
}

const getLayout = (plex: Mounted): string[] =>
  FAMILY.nodes.map((node) => plex.get(`[aria-label^="${node.title}"]`).attributes('transform') ?? '')

const getLayoutOf = async (props: Partial<InstanceType<typeof Plex>['$props']>) => {
  const plex = await mountPlex(props)
  const layout = getLayout(plex)
  plex.unmount()
  return layout
}

describe('the widgets of the plex that draw nothing', () => {
  it('keep the nodes where they are with no widgets at all', async () => {
    const bare = await getLayoutOf({})
    const nothing = ref(false)

    const plex = await mountPlex({
      widgetsAbove: [createEntry('parents', nothing)],
      widgetsBelow: [createEntry('tags', nothing)],
    })

    expect(getLayout(plex)).toStrictEqual(bare)
  })

  it('keep the nodes where they are when only some of the sides draw nothing', async () => {
    const drawing = ref(true)
    const nothing = ref(false)
    const onlyBelow = await getLayoutOf({ widgetsBelow: [createEntry('tags', drawing)] })

    const plex = await mountPlex({
      widgetsAbove: [createEntry('parents', nothing)],
      widgetsBelow: [createEntry('tags', drawing)],
    })

    expect(getLayout(plex)).toStrictEqual(onlyBelow)
  })

  it('move the nodes once, to where they are with the widget drawing from the start, when the widget starts drawing', async () => {
    const bare = await getLayoutOf({})
    const drawn = await getLayoutOf({ widgetsBelow: [createEntry('tags', ref(true))] })
    const drawing = ref(false)
    const plex = await mountPlex({ widgetsBelow: [createEntry('tags', drawing)] })
    const before = getLayout(plex)

    drawing.value = true
    await nextTick()
    await settle()

    expect(before).toStrictEqual(bare)
    expect(getLayout(plex)).not.toStrictEqual(bare)
    expect(getLayout(plex)).toStrictEqual(drawn)
  })

  it('give the room back when the last widget stops drawing', async () => {
    const bare = await getLayoutOf({})
    const drawing = ref(true)
    const plex = await mountPlex({ widgetsBelow: [createEntry('tags', drawing)] })
    expect(getLayout(plex)).not.toStrictEqual(bare)

    drawing.value = false
    await nextTick()
    await settle()

    expect(getLayout(plex)).toStrictEqual(bare)
  })

  it('keep the room while another widget of the side still draws', async () => {
    const tags = ref(true)
    const related = ref(true)
    const plex = await mountPlex({
      widgetsBelow: [createEntry('tags', tags), createEntry('related', related)],
    })
    const both = getLayout(plex)
    const onlyRelated = await getLayoutOf({ widgetsBelow: [createEntry('related', ref(true))] })

    tags.value = false
    await nextTick()
    await settle()

    expect(getLayout(plex)).toStrictEqual(onlyRelated)
    expect(getLayout(plex)).not.toStrictEqual(both)
  })
})
