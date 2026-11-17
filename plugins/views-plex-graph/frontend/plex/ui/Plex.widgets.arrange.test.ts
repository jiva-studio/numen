/** The plex arranges its nodes once the widgets they carry have arranged in size, not once for each widget that draws or measures on its own. */
import { flushPromises, mount } from '@vue/test-utils'
import { computed, defineComponent, h, nextTick, ref, watch, type Ref } from 'vue'
import { useSlotLoading } from '@numen/plugin-sdk'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import { arrangePlex, interpolatePlex, type Placement, type Size } from '../lib/arrange'
import { clusters } from '../lib/arrange/layout/clusters'
import type { Clock, Viewport } from '@numen/ui'
import type { PlexGraph, PlexNode } from '../../types'

vi.mock('../lib/arrange', async (importOriginal) => {
  const original = await importOriginal<typeof import('../lib/arrange')>()
  return {
    ...original,
    arrangePlex: vi.fn(original.arrangePlex),
    interpolatePlex: vi.fn(original.interpolatePlex),
  }
})

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
const FRAME = 16
const FRAMES_PER_ARRANGEMENT = 3

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

const deliverOneAfterAnother = async (arrivals: Arrivals, ids: readonly string[], draws = true) => {
  for (const id of ids) {
    deliver(arrivals, { [id]: draws })
    await nextTick()
  }
}

// The real placement, wrapped to count how often the plex asks the arrangement to place nodes.
const placed: number[] = []

const countingPlacement: Placement = {
  name: clusters.name,
  place: (nodes, focus, context) => {
    placed.push(placed.length)
    return clusters.place(nodes, focus, context)
  },
}

const mounted = new Map<string, number>()

const countMount = (name: string, path: string) =>
  mounted.set(`${path}|${name}`, (mounted.get(`${path}|${name}`) ?? 0) + 1)

const getMountedPaths = () => new Set([...mounted.keys()].map((key) => key.split('|')[0]))

const getEntryMounts = (id: string) =>
  [...mounted].filter(([key]) => key.startsWith(`${getPathOf(id)}|`)).map(([, count]) => count)

const createWidget = (name: string, arrivals: Arrivals) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    setup(props) {
      countMount(name, props.path)
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
  vi.mocked(arrangePlex).mockClear()
  vi.mocked(interpolatePlex).mockClear()
  mounted.clear()
  placed.length = 0
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
  // is as tall as its widgets, the plex adding nothing to it.
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const drawn =
      this.querySelectorAll('[data-widget]').length + (this.matches('[data-widget]') ? 1 : 0)
    const height = this.matches('.plex__widgets-group')
      ? drawn * WIDGET_HEIGHT
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
      placement: countingPlacement,
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

const getArrangedLayout = async (
  drawing: readonly string[],
  entries = (arrivals: Arrivals) => [createEntry(arrivals)],
  props: Partial<InstanceType<typeof Plex>['$props']> = {},
): Promise<string[]> => {
  const arrivals = createArrivals({ ...answerFor(ALL, false), ...answerFor(drawing) })
  const plex = mountPlex({ widgetsBelow: entries(arrivals), ...props })
  await passFrames(FRAMES_PER_ARRANGEMENT)
  const layout = getLayout(plex)
  plex.unmount()
  assigned = []
  return layout
}

const expectOnlyPositions = (arranged: readonly string[]) => {
  const positions = getPositions()
  FAMILY.nodes.forEach((node, index) => {
    expect(
      (positions.get(node.title) ?? []).filter((one) => one !== arranged[index]),
    ).toStrictEqual([])
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
  it('gives each node only the position the arranged widgets leave it, when the data arrives one note after another', async () => {
    const arranged = await getArrangedLayout(ALL)
    const arrivals = createArrivals()

    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getLayout(plex)).toStrictEqual(arranged)
    expectOnlyPositions(arranged)
  })

  it('draws no node while the widget of a visible node is still loading', async () => {
    const arrivals = createArrivals(answerFor(ALL.filter((id) => id !== 'child-1')))

    mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getAssignedNodes()).toBe(0)
  })

  it('places the nodes at the arranged layout, not moving them from one drawn before the widgets were there', async () => {
    const arranged = await getArrangedLayout(ALL)
    const arrivals = createArrivals()

    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)], duration: 420 })
    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getLayout(plex)).toStrictEqual(arranged)
  })

  it('arranges once for widgets that start drawing later in the nodes on screen, with no position between', async () => {
    const onlyFocus = ['focus']
    const arrangedBefore = await getArrangedLayout(onlyFocus)
    const arrangedAfter = await getArrangedLayout(ALL)
    const arrivals = createArrivals({ ...answerFor(ALL, false), ...answerFor(onlyFocus) })
    const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    expect(getLayout(plex)).toStrictEqual(arrangedBefore)
    assigned = []

    await deliverOneAfterAnother(arrivals, ['parent-0', 'parent-1'])
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(arrangedAfter).not.toStrictEqual(arrangedBefore)
    expect(getLayout(plex)).toStrictEqual(arrangedAfter)
    expectOnlyPositions(arrangedAfter)
  })

  it('never takes a node off the screen that it has drawn while the widgets load', async () => {
    const reference = mountPlex({
      widgetsBelow: [createEntry(createArrivals(answerFor(ALL)))],
      viewport: createViewport(SHORT_WINDOW),
    })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const arrangedTitles = getDrawnTitles(reference)
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
    await passFrames(FRAMES_PER_ARRANGEMENT)
    collect(observer.takeRecords())
    observer.disconnect()

    expect(arrangedTitles.length).toBeLessThan(FAMILY.nodes.length)
    expect(getDrawnTitles(plex)).toStrictEqual(arrangedTitles)
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
    const arrangedPlex = mountPlex({ widgetsBelow: both(reference) })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const arranged = getLayout(arrangedPlex)
    arrangedPlex.unmount()
    assigned = []
    const related = createArrivals(answerFor(ALL))
    const tags = createArrivals()

    const plex = mountPlex({ widgetsBelow: both([related, tags]) })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    expect(getAssignedNodes()).toBe(0)
    await deliverOneAfterAnother(tags, ALL)
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getLayout(plex)).toStrictEqual(arranged)
    expectOnlyPositions(arranged)
  })
})

describe('the plex with a widget that does not use the loading signal', () => {
  it('lays out once, at once the widget is measured', async () => {
    const reference = mountPlex({ widgetsBelow: [createPlainEntry()] })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const arranged = getLayout(reference)
    reference.unmount()
    assigned = []

    const plex = mountPlex({ widgetsBelow: [createPlainEntry()] })
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getLayout(plex)).toStrictEqual(arranged)
    expectOnlyPositions(arranged)
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
    await passFrames(FRAMES_PER_ARRANGEMENT)
    expect(getAssignedNodes()).toBe(0)

    await plex.setProps({ graph: withoutSpring })
    await passFrames(FRAMES_PER_ARRANGEMENT)

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

const createLateViewport = (size: Size): Viewport => {
  const deliver = (took: (size: Size) => void) => {
    let isLive = true
    Promise.resolve()
      .then(() => {
        if (isLive) took(size)
      })
      .catch(() => undefined)
    return () => {
      isLive = false
    }
  }
  return {
    watch: (_of, took) => deliver(took),
    watchFreeSpace: deliver,
    watchWhole: () => () => undefined,
  }
}

const CROWD: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    ...Array.from({ length: 3 }, (_, index) => ({
      id: `parent-${index}`,
      title: `Basin ${index}`,
      relation: 'parent' as const,
    })),
    ...Array.from({ length: 24 }, (_, index) => ({
      id: `child-${index}`,
      title: `Tributary ${index}`,
      relation: 'child' as const,
    })),
  ],
  edges: [
    ...Array.from({ length: 3 }, (_, index) => ({ from: `parent-${index}`, to: 'focus' })),
    ...Array.from({ length: 24 }, (_, index) => ({ from: 'focus', to: `child-${index}` })),
  ],
}

const CROWD_IDS = CROWD.nodes.map((node) => node.id)
const MEDIUM_WINDOW = { width: 600, height: 260 }

const getShownCount = (plex: Mounted) => plex.findAll('.plex__node[aria-label]').length

const getPlacedCount = () => placed.length

describe('the plex that waits for the widgets of many nodes', () => {
  it('arranges for the widget sizes of an open no more often than for widgets that all arrive at once', async () => {
    const together = createArrivals()
    const reference = mountPlex({ widgetsBelow: [createEntry(together)] })
    deliver(together, answerFor(ALL))
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const forOne = getPlacedCount()
    reference.unmount()
    placed.length = 0
    const arrivals = createArrivals()
    mountPlex({ widgetsBelow: [createEntry(arrivals)] })

    await deliverOneAfterAnother(arrivals, ALL)
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getPlacedCount()).toBe(forOne)
  })

  it('arranges nothing more while a widget still loads, however the sizes of the others change', async () => {
    const arrivals = createArrivals()
    mountPlex({ widgetsBelow: [createEntry(arrivals)] })
    await passFrames(1)
    const roomless = getPlacedCount()

    for (const id of ['focus', 'parent-0', 'parent-1', 'child-0']) {
      deliver(arrivals, { [id]: true })
      await passFrames(1)
    }

    expect(getPlacedCount()).toBe(roomless)
  })

  it('arranges once for the widgets that change size in one frame after the plex has arranged, as for one', async () => {
    const countForChanges = async (ids: readonly string[]) => {
      const arrivals = createArrivals(answerFor(ALL, false))
      const plex = mountPlex({ widgetsBelow: [createEntry(arrivals)] })
      await passFrames(FRAMES_PER_ARRANGEMENT)
      placed.length = 0
      await deliverOneAfterAnother(arrivals, ids)
      await passFrames(FRAMES_PER_ARRANGEMENT)
      plex.unmount()
      return getPlacedCount()
    }
    const forOne = await countForChanges(['parent-0'])

    expect(forOne).toBeGreaterThan(0)
    expect(await countForChanges(['parent-0', 'parent-1', 'child-0'])).toBe(forOne)
  })

  it('mounts the widgets of the nodes the window shows, and of no node that only the first pass offered', async () => {
    const bare = mountPlex({ graph: CROWD, viewport: createLateViewport(MEDIUM_WINDOW) })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const shownBare = getShownCount(bare)
    bare.unmount()
    mounted.clear()

    const arrivals = createArrivals(answerFor(CROWD_IDS))
    mountPlex({
      graph: CROWD,
      widgetsBelow: [createEntry(arrivals)],
      viewport: createLateViewport(MEDIUM_WINDOW),
    })
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(shownBare).toBeLessThan(CROWD.nodes.length)
    expect(getMountedPaths().size).toBeLessThanOrEqual(shownBare)
  })

  it('mounts the widgets of a node once while it stays on screen through two arrangements', async () => {
    const getShownIds = (plex: Mounted) =>
      CROWD.nodes
        .filter((node) => plex.find(`[aria-label^="${node.title}"]`).exists())
        .map((node) => node.id)
    const arrivals = createArrivals(answerFor(CROWD_IDS, false))
    const plex = mountPlex({
      graph: CROWD,
      widgetsBelow: [createEntry(arrivals)],
      viewport: createLateViewport(MEDIUM_WINDOW),
    })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const before = getShownIds(plex)

    deliver(arrivals, { focus: true })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    await plex.setProps({ graph: { ...CROWD, nodes: [...CROWD.nodes].reverse() } })
    await passFrames(FRAMES_PER_ARRANGEMENT)
    const staying = before.filter((id) => getShownIds(plex).includes(id))

    expect(staying).toContain('focus')
    expect(
      staying.filter(
        (id) => getEntryMounts(id).length === 0 || getEntryMounts(id).some((count) => count !== 1),
      ),
    ).toStrictEqual([])
  })
})

const getShownIdsOf = (graph: PlexGraph, plex: Mounted) =>
  graph.nodes
    .filter((node) => plex.find(`[aria-label^="${node.title}"]`).exists())
    .map((node) => node.id)

const mountCrowd = (arrivals: Arrivals, height: number, graph = CROWD) =>
  mountPlex({
    graph,
    widgetsBelow: [createEntry(arrivals, 'names'), createEntry(arrivals, 'tags')],
    viewport: createLateViewport({ width: 600, height }),
  })

const refocus = (graph: PlexGraph, id: string): PlexGraph => ({
  nodes: graph.nodes.map((node) => ({
    ...node,
    relation: node.id === id ? 'focus' : node.id === 'focus' ? 'parent' : node.relation,
  })),
  edges: [
    { from: 'focus', to: id },
    ...graph.edges.filter((edge) => edge.to !== 'focus' && edge.from !== 'focus'),
  ],
})

describe('the admission of the plex while the widgets of the shown nodes change its rooms', () => {
  it.each([220, 260, 300, 340, 380, 420])(
    'keeps the nodes it dropped dropped and mounts no node twice, in a window %i high',
    async (height) => {
      const arrivals = createArrivals(answerFor(CROWD_IDS))
      const plex = mountCrowd(arrivals, height)
      const shown: string[][] = []

      for (let frame = 0; frame < 14; frame += 1) {
        await passFrames(1)
        shown.push(getShownIdsOf(CROWD, plex))
      }

      const settled = shown.slice(2).map((ids) => ids.join())
      expect(new Set(settled).size).toBe(1)
      expect([...mounted].filter(([, count]) => count > 1)).toStrictEqual([])
    },
  )

  it('keeps every node it showed when the rooms of late widgets arrive after the emergency cap', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    const arrivals = createArrivals()
    const plex = mountCrowd(arrivals, 520)
    await passFrames(5, EMERGENCY_CAP / 4)
    const first = getShownIdsOf(CROWD, plex)

    await deliverOneAfterAnother(arrivals, CROWD_IDS)
    await passFrames(6)

    expect(first.length).toBeGreaterThan(1)
    expect(first.filter((id) => !getShownIdsOf(CROWD, plex).includes(id))).toStrictEqual([])
  })

  it('keeps the widgets of no more nodes in the document than the plex ends up showing', async () => {
    const arrivals = createArrivals(answerFor(CROWD_IDS))
    const plex = mountCrowd(arrivals, 380)

    await passFrames(10)

    expect(getShownCount(plex)).toBeLessThan(CROWD.nodes.length)
    expect(plex.findAll('.plex__widgets--below')).toHaveLength(getShownCount(plex))
  })

  it('arranges at most three times on a click and mounts each shown node once', async () => {
    const arrivals = createArrivals(answerFor(CROWD_IDS))
    const plex = mountCrowd(arrivals, 380)
    await passFrames(10)
    const forOpen = getPlacedCount()
    placed.length = 0
    mounted.clear()
    assigned = []

    await plex.setProps({ graph: refocus(CROWD, 'child-0') })
    await passFrames(10)

    expect(getPlacedCount()).toBeLessThanOrEqual(forOpen * 2)
    expect([...mounted].filter(([, count]) => count > 1)).toStrictEqual([])
    const given = new Map<Element, string[]>()
    for (const { element, value } of assigned) {
      if (!element.classList.contains('plex__node')) continue
      const values = given.get(element) ?? []
      if (values.at(-1) !== value) values.push(value)
      given.set(element, values)
    }
    expect(given.size).toBeGreaterThan(0)
    expect([...given.values()].filter((values) => values.length > 3)).toStrictEqual([])
  })
})

/** The arrangements that were given the rooms the widgets measured, and the movements that were started. */
const getArranged = () =>
  vi.mocked(arrangePlex).mock.calls.filter(([, input]) => input?.room !== undefined)

const getRuns = () => new Set(vi.mocked(interpolatePlex).mock.calls.map(([, to]) => to)).size

const mountMoving = (arrivals: Arrivals, height: number) =>
  mountPlex({
    graph: CROWD,
    widgetsBelow: [createEntry(arrivals, 'names'), createEntry(arrivals, 'tags')],
    viewport: createLateViewport({ width: 600, height }),
    duration: 420,
  })

describe('the plex that arranges once, with the rooms its widgets measured', () => {
  it('mounts the widgets of no node a second time on a click, while the nodes walk to their places', async () => {
    const plex = mountMoving(createArrivals(answerFor(CROWD_IDS)), 380)
    await passFrames(40)
    mounted.clear()

    await plex.setProps({ graph: refocus(CROWD, 'child-0') })
    await passFrames(40)

    expect(mounted.size).toBeGreaterThan(0)
    expect([...mounted].filter(([, count]) => count > 1)).toStrictEqual([])
  })

  it('arranges once with the measured rooms and starts one movement on a click', async () => {
    const plex = mountMoving(createArrivals(answerFor(CROWD_IDS)), 380)
    await passFrames(40)
    vi.mocked(arrangePlex).mockClear()
    vi.mocked(interpolatePlex).mockClear()

    await plex.setProps({ graph: refocus(CROWD, 'child-0') })
    await passFrames(40)

    expect(getArranged()).toHaveLength(1)
    expect(getRuns()).toBe(1)
  })

  it('draws the first layout with the notes the measured rooms admit, and draws nothing before it', async () => {
    const arrivals = createArrivals()
    const plex = mountMoving(arrivals, 380)
    const drawn: string[][] = []

    await deliverOneAfterAnother(arrivals, CROWD_IDS)
    for (let frame = 0; frame < 12; frame += 1) {
      await passFrames(1)
      drawn.push(getShownIdsOf(CROWD, plex))
    }

    const first = drawn.find((ids) => ids.length > 0)
    expect(first).toStrictEqual(drawn.at(-1))
    expect(first?.length).toBeLessThan(CROWD.nodes.length)
    expect(getArranged()).toHaveLength(1)
  })

  it('draws no note before the window has reported its size, for a plex without widgets as well', async () => {
    mountPlex({ graph: CROWD, viewport: createLateViewport(MEDIUM_WINDOW) })

    expect(getAssignedNodes()).toBe(0)
    expect(arrangePlex).not.toHaveBeenCalled()
    await passFrames(FRAMES_PER_ARRANGEMENT)

    expect(getAssignedNodes()).toBeGreaterThan(0)
    expect(
      vi.mocked(arrangePlex).mock.calls.map(([, input]) => input?.options?.viewport),
    ).toStrictEqual(vi.mocked(arrangePlex).mock.calls.map(() => MEDIUM_WINDOW))
  })

  it('gives a note without widgets no room, and a note with widgets exactly the room they drew', async () => {
    const arrivals = createArrivals({ ...answerFor(CROWD_IDS, false), focus: true })
    mountPlex({
      graph: CROWD,
      widgetsBelow: [createEntry(arrivals)],
      viewport: createLateViewport({ width: 600, height: 380 }),
    })

    await passFrames(10)

    const room = getArranged().at(-1)?.[1]?.room
    expect(room?.(CROWD.nodes[0] as PlexNode)).toMatchObject({ top: 0, bottom: WIDGET_HEIGHT })
    expect(room?.(CROWD.nodes[1] as PlexNode)).toMatchObject({ top: 0, bottom: 0 })
  })

  it('re-arranges once for the widgets that grow in one frame, and shows the same notes', async () => {
    const names = createArrivals(answerFor(CROWD_IDS))
    const tags = createArrivals(answerFor(CROWD_IDS, false))
    const plex = mountPlex({
      graph: CROWD,
      widgetsBelow: [createEntry(names, 'names'), createEntry(tags, 'tags')],
      viewport: createLateViewport({ width: 600, height: 380 }),
      duration: 420,
    })
    await passFrames(40)
    const before = getShownIdsOf(CROWD, plex)
    vi.mocked(arrangePlex).mockClear()

    deliver(tags, answerFor(['parent-0', 'parent-1', 'child-0']))
    await passFrames(40)

    expect(getArranged()).toHaveLength(1)
    expect(getShownIdsOf(CROWD, plex)).toStrictEqual(before)
  })
})
