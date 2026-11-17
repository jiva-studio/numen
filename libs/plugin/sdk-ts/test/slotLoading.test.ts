/** A view of a slot says when its data has arrived, and the owner of the slot waits for every view it drew. */
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref, type Component } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  SLOT_LOADING_CAP_MS,
  SLOT_LOADING_KEY,
  createSlotLoading,
  provideSlotLoading,
  useSlotLoading,
  type SlotLoadingClock,
  type SlotLoadingEntry,
} from '../src'

const LEDGER: SlotLoadingEntry = {
  key: 'org.example.ledger/view/related',
  pluginId: 'org.example.ledger',
}
const HARBOR: SlotLoadingEntry = {
  key: 'org.example.harbor/view/tags',
  pluginId: 'org.example.harbor',
}

let scheduled: ((now: number) => void)[] = []
let now = 0

const clock: SlotLoadingClock = {
  now: () => now,
  schedule: (run) => scheduled.push(run),
  cancel: () => undefined,
}

const passTime = (ms: number) => {
  now += ms
  const waiting = scheduled
  scheduled = []
  for (const run of waiting) run(now)
}

afterEach(() => {
  scheduled = []
  now = 0
})

/** A view that is loading until the test calls the finish it was given. */
const createLoadingView = (finishes: (() => void)[]): Component =>
  defineComponent({
    setup() {
      finishes.push(useSlotLoading())
      return () => h('span', 'view')
    },
  })

const createEntry = (
  loading: ReturnType<typeof createSlotLoading>,
  entry: SlotLoadingEntry,
  view: Component,
) =>
  defineComponent({
    setup() {
      provideSlotLoading(loading, entry)
      return () => h(view)
    },
  })

describe('useSlotLoading', () => {
  it('is a no-op outside the host of a slot', () => {
    let finish: (() => void) | undefined
    const view = defineComponent({
      setup() {
        finish = useSlotLoading()
        return () => h('span')
      },
    })

    mount(view)

    expect(finish).toBeTypeOf('function')
    expect(() => {
      finish?.()
      finish?.()
    }).not.toThrow()
  })

  it('registers the view with its host when it is set up, and releases it once however often it finishes', () => {
    const release = vi.fn()
    const begin = vi.fn(() => release)
    const finishes: (() => void)[] = []

    mount(createLoadingView(finishes), {
      global: { provide: { [SLOT_LOADING_KEY as symbol]: { begin } } },
    })

    expect(begin).toHaveBeenCalledTimes(1)
    expect(release).not.toHaveBeenCalled()
    finishes[0]?.()
    finishes[0]?.()
    expect(release).toHaveBeenCalledTimes(1)
  })
})

describe('the loading of a slot', () => {
  const mountSlot = (
    entries: readonly { entry: SlotLoadingEntry; view: Component }[],
    reportIssue = vi.fn(),
  ) => {
    const loading = createSlotLoading({ clock, reportIssue })
    const shown = ref(entries)
    const wrapper = mount(
      defineComponent({
        setup: () => () =>
          shown.value.map((one) =>
            h(createEntry(loading, one.entry, one.view), { key: one.entry.key }),
          ),
      }),
    )
    return { loading, wrapper, shown, reportIssue }
  }

  it('is ready with no view', () => {
    expect(mountSlot([]).loading.isReady.value).toBe(true)
  })

  it('is ready at once when no view of it calls useSlotLoading', () => {
    const plain = defineComponent({ setup: () => () => h('span') })

    expect(mountSlot([{ entry: LEDGER, view: plain }]).loading.isReady.value).toBe(true)
  })

  it('is not ready while a view loads, and is ready when it finishes', async () => {
    const finishes: (() => void)[] = []
    const { loading } = mountSlot([{ entry: LEDGER, view: createLoadingView(finishes) }])
    expect(loading.isReady.value).toBe(false)

    finishes[0]?.()
    await nextTick()

    expect(loading.isReady.value).toBe(true)
  })

  it('is ready only when the views of every entry have finished, each entry holding its own', async () => {
    const related: (() => void)[] = []
    const tags: (() => void)[] = []
    const { loading } = mountSlot([
      { entry: LEDGER, view: createLoadingView(related) },
      { entry: HARBOR, view: createLoadingView(tags) },
    ])

    related[0]?.()
    await nextTick()
    expect(loading.isReady.value).toBe(false)

    tags[0]?.()
    await nextTick()
    expect(loading.isReady.value).toBe(true)
  })

  it('keeps waiting for another view when one finishes twice', async () => {
    const related: (() => void)[] = []
    const tags: (() => void)[] = []
    const { loading } = mountSlot([
      { entry: LEDGER, view: createLoadingView(related) },
      { entry: HARBOR, view: createLoadingView(tags) },
    ])

    related[0]?.()
    related[0]?.()
    await nextTick()

    expect(loading.isReady.value).toBe(false)
  })

  it('does not wait for an entry that is unmounted', async () => {
    const related: (() => void)[] = []
    const tags: (() => void)[] = []
    const { loading, shown } = mountSlot([
      { entry: LEDGER, view: createLoadingView(related) },
      { entry: HARBOR, view: createLoadingView(tags) },
    ])
    related[0]?.()

    shown.value = shown.value.filter((one) => one.entry !== HARBOR)
    await nextTick()

    expect(loading.isReady.value).toBe(true)
  })

  it('releases a view that has not finished once the emergency cap has passed, and reports it by name', async () => {
    const { loading, reportIssue } = mountSlot([{ entry: LEDGER, view: createLoadingView([]) }])

    passTime(SLOT_LOADING_CAP_MS - 1)
    await nextTick()
    expect(loading.isReady.value).toBe(false)
    expect(reportIssue).not.toHaveBeenCalled()

    passTime(2)
    await nextTick()

    expect(loading.isReady.value).toBe(true)
    expect(reportIssue).toHaveBeenCalledTimes(1)
    expect(reportIssue).toHaveBeenCalledWith(
      LEDGER.pluginId,
      expect.stringContaining('did not report ready'),
    )
    expect(reportIssue.mock.calls[0]?.[1]).toContain(LEDGER.key)
  })

  it('reports nothing for a view that finishes in time', async () => {
    const finishes: (() => void)[] = []
    const { reportIssue } = mountSlot([{ entry: LEDGER, view: createLoadingView(finishes) }])

    finishes[0]?.()
    passTime(SLOT_LOADING_CAP_MS * 2)
    await nextTick()

    expect(reportIssue).not.toHaveBeenCalled()
  })
})
