/** The plex arranges its nodes once the widgets they carry have settled in size, not once for each widget that draws or measures on its own. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref, type Ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import type { Size } from '../lib/arrange'
import type { Clock, Viewport } from '@numen/ui'
import type { PlexGraph } from '../../types'

const FAMILY: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    { id: 'parent-0', title: 'Hydrology', relation: 'parent' },
    { id: 'parent-1', title: 'Geology', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'child' },
    { id: 'child-1', title: 'Spring', relation: 'child' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'parent-1', to: 'focus' },
    { from: 'focus', to: 'child-0' },
    { from: 'focus', to: 'child-1' },
  ],
}

const WIDGET_HEIGHT = 40
const NODE_GAP = 8
const FRAME = 16
const FRAMES_PER_SETTLE = 3

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

const getPathOf = (id: string) => `notes/${id}.md`

const createWidget = (name: string, drawing: Readonly<Ref<ReadonlySet<string>>>) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    setup: (props) => () =>
      drawing.value.has(props.path) ? h('span', { 'data-widget': name }, props.path) : null,
  })

const createEntry = (drawing: Readonly<Ref<ReadonlySet<string>>>) => ({
  key: 'org.example.tags/view/tags',
  component: createWidget('tags', drawing),
})

const getPaths = (ids: readonly string[]): ReadonlySet<string> => new Set(ids.map(getPathOf))

const ALL = FAMILY.nodes.map((node) => node.id)

const drawOneAfterAnother = async (
  drawing: Ref<ReadonlySet<string>>,
  ids: readonly string[],
): Promise<void> => {
  for (const id of ids) {
    drawing.value = new Set([...drawing.value, getPathOf(id)])
    await nextTick()
  }
}

let rings: (() => void)[] = []
let assigned: { element: Element; value: string }[] = []
let scheduled: ((now: number) => void)[] = []
let now = 0

const clock: Clock = {
  now: () => now,
  schedule: (run) => scheduled.push(run),
  cancel: () => undefined,
}

beforeEach(() => {
  rings = []
  assigned = []
  scheduled = []
  now = 0
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
    const drawn =
      this.querySelectorAll('[data-widget]').length + (this.matches('[data-widget]') ? 1 : 0)
    const height = this.matches('.plex__widgets-group')
      ? NODE_GAP + drawn * WIDGET_HEIGHT
      : drawn > 0
        ? WIDGET_HEIGHT
        : 0
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
  // Every position a node is given is kept, the ones drawn for a moment as well as the last.
  const setAttribute = Element.prototype.setAttribute
  vi.spyOn(Element.prototype, 'setAttribute').mockImplementation(function (
    this: Element,
    name: string,
    value: string,
  ) {
    if (name === 'transform') assigned.push({ element: this, value })
    setAttribute.call(this, name, value)
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

const passFrames = async (count: number) => {
  for (let frame = 0; frame < count; frame += 1) {
    await flushPromises()
    for (const ring of rings) ring()
    await flushPromises()
    now += FRAME
    const waiting = scheduled
    scheduled = []
    for (const run of waiting) run(now)
    await new Promise((resolve) => setTimeout(resolve, FRAME))
  }
  await flushPromises()
}

type Mounted = ReturnType<typeof mountPlex>

const mountPlex = (props: Partial<InstanceType<typeof Plex>['$props']> = {}) =>
  mount(Plex, {
    props: {
      graph: FAMILY,
      duration: 0,
      clock,
      viewport: createViewport({ width: 1200, height: 1200 }),
      getPath: getPathOf,
      ...props,
    },
    attachTo: document.body,
  })

const getTitleOf = (element: Element): string | undefined =>
  FAMILY.nodes.find((node) => element.getAttribute('aria-label')?.startsWith(node.title))?.title

const getPositions = (): Map<string, string[]> => {
  const found = new Map<string, string[]>()
  for (const { element, value } of assigned) {
    const title = element.classList.contains('plex__node') ? getTitleOf(element) : undefined
    if (!title) continue
    const given = found.get(title) ?? []
    if (given.at(-1) !== value) given.push(value)
    found.set(title, given)
  }
  return found
}

const getLayout = (plex: Mounted) =>
  FAMILY.nodes.map(
    (node) => plex.get(`[aria-label^="${node.title}"]`).attributes('transform') ?? '',
  )

const getSettledLayout = async (ids: readonly string[]): Promise<string[]> => {
  const plex = mountPlex({ widgetsBelow: [createEntry(ref(getPaths(ids)))] })
  await passFrames(FRAMES_PER_SETTLE)
  const layout = getLayout(plex)
  plex.unmount()
  assigned = []
  return layout
}

const expectOnlyPositions = (settled: readonly string[]) => {
  const positions = getPositions()
  FAMILY.nodes.forEach((node, index) => {
    expect((positions.get(node.title) ?? []).filter((one) => one !== settled[index])).toStrictEqual(
      [],
    )
  })
}

describe('the plex whose widgets draw after the nodes do', () => {
  it('gives each node only the position the settled widgets leave it, when the widgets draw one after another', async () => {
    const settled = await getSettledLayout(ALL)
    const drawing = ref<ReadonlySet<string>>(new Set())

    const plex = mountPlex({ widgetsBelow: [createEntry(drawing)] })
    await drawOneAfterAnother(drawing, ALL)
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
    expectOnlyPositions(settled)
  })

  it('places the nodes at the settled layout, not moving them from one drawn before the widgets were there', async () => {
    const settled = await getSettledLayout(ALL)
    const drawing = ref<ReadonlySet<string>>(new Set())

    const plex = mountPlex({ widgetsBelow: [createEntry(drawing)], duration: 420 })
    await drawOneAfterAnother(drawing, ALL)
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
  })

  it('arranges once for widgets that start drawing later in the nodes on screen, with no position between', async () => {
    const onlyFocus = ['focus']
    const settledBefore = await getSettledLayout(onlyFocus)
    const settledAfter = await getSettledLayout(ALL)
    const drawing = ref(getPaths(onlyFocus))
    const plex = mountPlex({ widgetsBelow: [createEntry(drawing)] })
    await passFrames(FRAMES_PER_SETTLE)
    expect(getLayout(plex)).toStrictEqual(settledBefore)
    assigned = []

    await drawOneAfterAnother(drawing, ['parent-0', 'parent-1'])
    await passFrames(FRAMES_PER_SETTLE)

    expect(settledAfter).not.toStrictEqual(settledBefore)
    expect(getLayout(plex)).toStrictEqual(settledAfter)
    expectOnlyPositions(settledAfter)
  })
})
