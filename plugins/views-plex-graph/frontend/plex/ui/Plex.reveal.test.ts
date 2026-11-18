/** A click on a node changes what the plex shows in one step: the nodes, their titles and their widgets stay as they were until the next arrangement is ready, then they appear and move together. */
import { flushPromises, mount } from '@vue/test-utils'
import { computed, defineComponent, h, markRaw, nextTick, ref, watch, type Ref } from 'vue'
import { useSlotLoading } from '@numen/plugin-sdk'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import type { Size } from '../lib/arrange'
import type { Clock, Viewport } from '@numen/ui'
import type { PlexGraph } from '../../types'

const WIDGET_HEIGHT = 40
const FRAME = 16
const MOVEMENT = 420

const BEFORE: PlexGraph = {
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

/** The delta is chosen, and the nodes come in another order: the watershed becomes its parent, the hydrology and the spring leave, and two notes of the delta come in. */
const AFTER: PlexGraph = {
  nodes: [
    { id: 'estuary', title: 'Estuary', relation: 'child' },
    { id: 'silt', title: 'Silt', relation: 'child' },
    { id: 'focus', title: 'Watershed', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'focus' },
  ],
  edges: [
    { from: 'focus', to: 'child-0' },
    { from: 'child-0', to: 'silt' },
    { from: 'child-0', to: 'estuary' },
  ],
}

const STAYING = ['focus', 'child-0']
const ENTERING = ['silt', 'estuary']
const LEAVING = ['parent-0', 'child-1']
const TITLES = new Map([...BEFORE.nodes, ...AFTER.nodes].map((node) => [node.id, node.title]))

const getPathOf = (id: string) => `notes/${id}.md`
const getWidgetText = (id: string) => `names of ${id}`

type Arrivals = Ref<ReadonlySet<string>>

const createWidget = (arrivals: Arrivals) =>
  markRaw(
    defineComponent({
      props: { path: { type: String, required: true } },
      setup(props) {
        const finish = useSlotLoading()
        const hasArrived = computed(() => arrivals.value.has(props.path))
        watch(
          hasArrived,
          (arrived) => {
            if (arrived) nextTick(finish).catch(() => undefined)
          },
          { immediate: true },
        )
        return () =>
          hasArrived.value
            ? h(
                'span',
                { 'data-widget': 'names' },
                props.path.replace(/^notes\/(.*)\.md$/, 'names of $1'),
              )
            : null
      },
    }),
  )

const deliver = (arrivals: Arrivals, ids: readonly string[]) => {
  arrivals.value = new Set([...arrivals.value, ...ids.map(getPathOf)])
}

const createViewport = (size: Size): Viewport => {
  const watchSize = (took: (size: Size) => void) => {
    took(size)
    return () => undefined
  }
  return {
    watch: (_of, took) => watchSize(took),
    watchFreeSpace: watchSize,
    watchWhole: () => () => undefined,
  }
}

let rings: (() => void)[] = []
let scheduled: ((now: number) => void)[] = []
let now = 0

const clock: Clock = {
  now: () => now,
  schedule: (run) => scheduled.push(run),
  cancel: () => undefined,
}

beforeEach(() => {
  rings = []
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

/** What the reader sees of one node: its title, where it is, how opaque it is and the words of its widgets, for a node that is not held back. */
interface Seen {
  readonly title: string
  readonly at: string
  readonly opacity: string
  readonly widgets: readonly string[]
  readonly widgetOpacity: readonly string[]
}

const readVisibleNodes = (plex: ReturnType<typeof mountPlex>): Map<string, Seen> => {
  const seen = new Map<string, Seen>()
  for (const stack of plex.findAll('.plex__stack')) {
    if (stack.classes('plex__stack--waiting')) continue
    const node = stack.find('.plex__node')
    const label = node.exists() ? (node.attributes('aria-label') ?? '') : ''
    const id = [...TITLES].find(([, title]) => label.startsWith(title))?.[0]
    if (!id) continue
    const widgets = stack.findAll('.plex__widgets')
    seen.set(id, {
      title: node.text(),
      at: node.attributes('transform') ?? '',
      opacity: node.attributes('opacity') ?? '1',
      widgets: stack.findAll('[data-widget]').map((one) => one.text()),
      widgetOpacity: widgets.map((one) => one.attributes('opacity') ?? '1'),
    })
  }
  return seen
}

const mountPlex = (arrivals: Arrivals) =>
  mount(Plex, {
    props: {
      graph: BEFORE,
      duration: MOVEMENT,
      clock,
      viewport: createViewport({ width: 1200, height: 900 }),
      getPath: getPathOf,
      widgetsBelow: [{ key: 'org.example.names/view/names', component: createWidget(arrivals) }],
    },
    attachTo: document.body,
  })

const mountBeforeClick = async () => {
  const arrivals: Arrivals = ref(new Set())
  const plex = mountPlex(arrivals)
  deliver(
    arrivals,
    BEFORE.nodes.map((node) => node.id),
  )
  await passFrames(30)
  const resting = readVisibleNodes(plex)
  await plex.setProps({ graph: AFTER })
  return { arrivals, plex, resting }
}

const hasMoved = (resting: Map<string, Seen>, seen: Map<string, Seen>) =>
  [...seen].some(([id, one]) => resting.has(id) && resting.get(id)?.at !== one.at)

describe('the plex between a click and the frame of the new graph', () => {
  it('keeps every node, title and widget as they were while the widgets of the new nodes are loading', async () => {
    const { plex, resting } = await mountBeforeClick()

    await passFrames(30)

    expect(readVisibleNodes(plex)).toStrictEqual(resting)
    expect([...resting.keys()].sort()).toStrictEqual([...STAYING, ...LEAVING].sort())
    expect(resting.get('child-0')?.widgets).toStrictEqual([getWidgetText('child-0')])
  })

  it('starts nothing before the new frame is ready', async () => {
    const { plex } = await mountBeforeClick()

    await passFrames(30)

    expect(plex.find('[data-moving]').exists()).toBe(false)
  })
})

describe('the first frame of the movement to the new graph', () => {
  const run = async () => {
    const { arrivals, plex, resting } = await mountBeforeClick()
    await passFrames(10)
    deliver(arrivals, ENTERING)
    const frames: Map<string, Seen>[] = []
    for (let frame = 0; frame < 40; frame += 1) {
      await passFrames(1)
      frames.push(readVisibleNodes(plex))
    }
    const first = frames.findIndex((seen) => hasMoved(resting, seen))
    return { frames, first, resting }
  }

  it('shows the entering nodes with their widgets, as visible as the nodes', async () => {
    const { frames, first } = await run()

    expect(first).toBeGreaterThanOrEqual(0)
    for (const id of ENTERING) {
      const seen = frames[first]?.get(id)
      expect(seen?.widgets, id).toStrictEqual([getWidgetText(id)])
      expect(seen?.opacity, id).toBe('1')
      expect(seen?.widgetOpacity, id).toStrictEqual(['1'])
    }
  })

  it('keeps the title and the widgets of every node that was there for as long as it stays', async () => {
    const { frames, resting } = await run()

    for (const seen of frames) {
      for (const [id, one] of seen) {
        const was = resting.get(id)
        if (was === undefined) continue
        expect(one.title, id).toBe(was.title)
        expect(one.widgets, id).toStrictEqual(was.widgets)
      }
    }
  })

  it('shows the new nodes at once and none of them hidden or without its widget', async () => {
    const { frames, first } = await run()

    for (const seen of frames.slice(first)) {
      for (const id of [...STAYING, ...ENTERING]) {
        expect(seen.get(id)?.widgets, id).toStrictEqual([getWidgetText(id)])
      }
    }
  })
})

describe('the elements of the nodes through a click', () => {
  const trackElements = (plex: ReturnType<typeof mountPlex>) => {
    const titles = new Map<Element, Set<string>>()
    const places = new Map<Element, string[]>()
    const take = () => {
      for (const element of plex.element.querySelectorAll('.plex__node')) {
        const title = (element.textContent ?? '').trim()
        titles.set(element, (titles.get(element) ?? new Set()).add(title))
        const given = places.get(element) ?? []
        const at = element.getAttribute('transform') ?? ''
        if (given.at(-1) !== at) given.push(at)
        places.set(element, given)
      }
    }
    return { titles, places, take }
  }

  it('show one note each for as long as they live, and a staying node is the same element', async () => {
    const { arrivals, plex } = await mountBeforeClick()
    const before = new Map(
      [...plex.element.querySelectorAll('.plex__node')].map((one) => [
        one.getAttribute('aria-label')?.split(',')[0] ?? '',
        one,
      ]),
    )
    const tracked = trackElements(plex)
    deliver(arrivals, ENTERING)
    for (let frame = 0; frame < 40; frame += 1) {
      await passFrames(1)
      tracked.take()
    }

    expect([...tracked.titles.values()].filter((titles) => titles.size > 1)).toStrictEqual([])
    for (const id of STAYING) {
      const element = before.get(TITLES.get(id) ?? '')
      expect(plex.element.contains(element ?? null), id).toBe(true)
    }
  })

  it('moves an entering node from the clicked node to the place it ends at', async () => {
    const { arrivals, plex } = await mountBeforeClick()
    const tracked = trackElements(plex)
    deliver(arrivals, ENTERING)
    for (let frame = 0; frame < 40; frame += 1) {
      await passFrames(1)
      tracked.take()
    }

    const entering = [...tracked.places].filter(([element]) =>
      ENTERING.some((id) => (element.textContent ?? '').includes(TITLES.get(id) ?? '')),
    )
    expect(entering).toHaveLength(ENTERING.length)
    expect(entering.some(([, places]) => places.length > 1)).toBe(true)
  })
})
