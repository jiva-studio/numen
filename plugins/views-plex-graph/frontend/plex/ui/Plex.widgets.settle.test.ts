/** The plex arranges its nodes once the widgets they carry have settled in size, not once for each widget that draws or measures on its own. */
import { flushPromises, mount } from '@vue/test-utils'
import { computed, defineComponent, h, nextTick, ref, watch, type Ref } from 'vue'
import { useSlotLoading } from '@numen/plugin-sdk'
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

/** What the store has delivered for each note: whether its widget draws, for the notes it has answered. */
type Arrivals = Ref<ReadonlyMap<string, boolean>>

const createArrivals = (answers: Record<string, boolean> = {}): Arrivals =>
  ref(new Map(Object.entries(answers).map(([id, draws]) => [getPathOf(id), draws])))

const answerFor = (ids: readonly string[], draws = true): Record<string, boolean> =>
  Object.fromEntries(ids.map((id) => [id, draws]))

const deliver = (arrivals: Arrivals, answers: Record<string, boolean>) => {
  arrivals.value = new Map([
    ...arrivals.value,
    ...Object.entries(answers).map(([id, draws]): [string, boolean] => [getPathOf(id), draws]),
  ])
}

const ALL = FAMILY.nodes.map((node) => node.id)

/** Delivers the notes one tick after another, as the data of each note arrives from its own request. */
const deliverOneAfterAnother = async (arrivals: Arrivals, ids: readonly string[], draws = true) => {
  for (const id of ids) {
    deliver(arrivals, { [id]: draws })
    await nextTick()
  }
}

/** Loading until its note is delivered, then it draws, if its note says so, and reports ready. */
const createWidget = (name: string, arrivals: Arrivals) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    setup(props) {
      const finish = useSlotLoading()
      const draws = computed(() => arrivals.value.get(props.path))
      watch(
        draws,
        (answered) => {
          if (answered !== undefined) nextTick(finish).catch(() => undefined)
        },
        { immediate: true },
      )
      return () => (draws.value ? h('span', { 'data-widget': name }, props.path) : null)
    },
  })

/** Draws at once and never says anything about loading. */
const createPlainWidget = (name: string) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    setup: (props) => () => h('span', { 'data-widget': name }, props.path),
  })

const createEntry = (arrivals: Arrivals, name = 'tags') => ({
  key: `org.example.${name}/view/${name}`,
  component: createWidget(name, arrivals),
})

const createPlainEntry = (name = 'tags') => ({
  key: `org.example.${name}/view/${name}`,
  component: createPlainWidget(name),
})

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

const passFrames = async (count: number, step = FRAME) => {
  for (let frame = 0; frame < count; frame += 1) {
    await flushPromises()
    for (const ring of rings) ring()
    await flushPromises()
    now += step
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
  FAMILY.nodes.map((node) => {
    const found = plex.find(`[aria-label^="${node.title}"]`)
    return found.exists() ? (found.attributes('transform') ?? '') : ''
  })

const getSettledLayout = async (
  drawing: readonly string[],
  entries = (arrivals: Arrivals) => [createEntry(arrivals)],
  props: Partial<InstanceType<typeof Plex>['$props']> = {},
): Promise<string[]> => {
  const arrivals = createArrivals({ ...answerFor(ALL, false), ...answerFor(drawing) })
  const plex = mountPlex({ widgetsBelow: entries(arrivals), ...props })
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

const getAssignedNodes = () =>
  assigned.filter(({ element }) => element.classList.contains('plex__node')).length

const SHORT_WINDOW = { width: 1200, height: 300 }
/** The emergency cap of the loading of a view, which only a view that hangs reaches. */
const EMERGENCY_CAP = 1000

const getDrawnTitles = (plex: Mounted): string[] =>
  FAMILY.nodes
    .filter((node) => plex.find(`[aria-label^="${node.title}"]`).exists())
    .map((node) => node.title)

describe('the plex whose widgets load after the nodes are mounted', () => {
  it('gives each node only the position the settled widgets leave it, when the data arrives one note after another', async () => {
    const settled = await getSettledLayout(ALL)
    const arrivals = createArrivals()

    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
    expectOnlyPositions(settled)
  })

  it('draws no node while the widget of a visible node is still loading', async () => {
    const arrivals = createArrivals(answerFor(ALL.filter((id) => id !== 'child-1')))

    mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(FRAMES_PER_SETTLE)

    expect(getAssignedNodes()).toBe(0)
  })

  it('places the nodes at the settled layout, not moving them from one drawn before the widgets were there', async () => {
    const settled = await getSettledLayout(ALL)
    const arrivals = createArrivals()

    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)], duration: 420 })
    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
  })

  it('arranges once for widgets that start drawing later in the nodes on screen, with no position between', async () => {
    const onlyFocus = ['focus']
    const settledBefore = await getSettledLayout(onlyFocus)
    const settledAfter = await getSettledLayout(ALL)
    const arrivals = createArrivals({ ...answerFor(ALL, false), ...answerFor(onlyFocus) })
    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(FRAMES_PER_SETTLE)
    expect(getLayout(plex)).toStrictEqual(settledBefore)
    assigned = []

    await deliverOneAfterAnother(arrivals, ['parent-0', 'parent-1'])
    await passFrames(FRAMES_PER_SETTLE)

    expect(settledAfter).not.toStrictEqual(settledBefore)
    expect(getLayout(plex)).toStrictEqual(settledAfter)
    expectOnlyPositions(settledAfter)
  })

  it('never takes a node off the screen that it has drawn while the widgets settle', async () => {
    const reference = mountPlex({
      widgetsBelow: [createEntry(createArrivals(answerFor(ALL)))],
      viewport: createViewport(SHORT_WINDOW),
    })
    await passFrames(FRAMES_PER_SETTLE)
    const settledTitles = getDrawnTitles(reference)
    reference.unmount()
    const removed: string[] = []
    const collect = (records: MutationRecord[]) => {
      for (const record of records) {
        // A node that no longer fits is kept as a ghost without a label, so it is gone to the reader.
        if (
          record.attributeName === 'aria-label' &&
          !(record.target as Element).hasAttribute('aria-label')
        )
          removed.push(record.oldValue ?? '')
        for (const node of record.removedNodes)
          if (node instanceof Element)
            for (const gone of node.querySelectorAll('.plex__node'))
              removed.push(getTitleOf(gone) ?? '')
      }
    }
    const observer = new MutationObserver(collect)
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['aria-label'],
      attributeOldValue: true,
    })
    const arrivals = createArrivals()

    const plex = mountPlex({
      widgetsBelow: [createEntry(arrivals)],
      viewport: createViewport(SHORT_WINDOW),
    })
    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_SETTLE)
    collect(observer.takeRecords())
    observer.disconnect()

    expect(settledTitles.length).toBeLessThan(FAMILY.nodes.length)
    expect(getDrawnTitles(plex)).toStrictEqual(settledTitles)
    expect(removed).toStrictEqual([])
  })
})

describe('the plex whose slot has two contributors', () => {
  it('lays nothing out until both have delivered, then once', async () => {
    const both = (arrivals: Arrivals[]) => [
      createEntry(arrivals[0] as Arrivals, 'related'),
      createEntry(arrivals[1] as Arrivals, 'tags'),
    ]
    const reference = [createArrivals(answerFor(ALL)), createArrivals(answerFor(ALL))]
    const settledPlex = mountPlex({ widgetsBelow: both(reference) })
    await passFrames(FRAMES_PER_SETTLE)
    const settled = getLayout(settledPlex)
    settledPlex.unmount()
    assigned = []
    const related = createArrivals(answerFor(ALL))
    const tags = createArrivals()

    const plex = mountPlex({ widgetsBelow: both([related, tags]) })
    await passFrames(FRAMES_PER_SETTLE)
    expect(getAssignedNodes()).toBe(0)
    await deliverOneAfterAnother(tags, ALL)
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
    expectOnlyPositions(settled)
  })
})

describe('the plex with a widget that does not use the loading signal', () => {
  it('lays out once, at once the widget is measured', async () => {
    const reference = mountPlex({ widgetsBelow: [createPlainEntry()] })
    await passFrames(FRAMES_PER_SETTLE)
    const settled = getLayout(reference)
    reference.unmount()
    assigned = []

    const plex = mountPlex({ widgetsBelow: [createPlainEntry()] })
    await passFrames(FRAMES_PER_SETTLE)

    expect(getLayout(plex)).toStrictEqual(settled)
    expectOnlyPositions(settled)
  })
})

describe('the plex whose widget of a node leaves', () => {
  it('does not wait for a node that is no longer shown', async () => {
    const arrivals = createArrivals(answerFor(ALL.filter((id) => id !== 'child-1')))
    const withoutSpring: PlexGraph = {
      nodes: FAMILY.nodes.filter((node) => node.id !== 'child-1'),
      edges: FAMILY.edges.filter((edge) => edge.to !== 'child-1'),
    }
    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(FRAMES_PER_SETTLE)
    expect(getAssignedNodes()).toBe(0)

    await plex.setProps({ graph: withoutSpring })
    await passFrames(FRAMES_PER_SETTLE)

    expect(getDrawnTitles(plex)).toStrictEqual(withoutSpring.nodes.map((node) => node.title))
  })
})

describe('the plex whose widget never reports ready', () => {
  it('lays out with what it has once the emergency cap has passed, and warns by the name of the widget', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    const arrivals = createArrivals(answerFor(ALL.filter((id) => id !== 'child-1')))
    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals, 'hung')] })

    await passFrames(3, EMERGENCY_CAP / 4)
    expect(getAssignedNodes()).toBe(0)
    await passFrames(3, EMERGENCY_CAP / 4)

    expect(getDrawnTitles(plex)).toStrictEqual(FAMILY.nodes.map((node) => node.title))
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('did not report ready'))
    expect(warn.mock.calls.flat().join(' ')).toContain('org.example.hung/view/hung')
  })
})
