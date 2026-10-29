import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Reader from './Reader.vue'
import { READER_WORDS } from '../lib/strip'

const PAGES = Array.from({ length: 8 }, () => ({ width: 612, height: 792 }))

/**
 * A reader in a container of a given size. Tests have no layout, so the
 * container size is set explicitly and the reader is triggered to measure again.
 */
const reader = async (width: number, high: number) => {
  const wrapper = mount(Reader, {
    props: { pages: PAGES, picture: (page: number) => `/p/${page}` },
    attachTo: document.body,
  })
  await room(wrapper, width, high)
  return wrapper
}

const room = async (wrapper: ReturnType<typeof mount>, width: number, height: number) => {
  const area = wrapper.find('.reader__viewport').element as HTMLElement
  // jsdom lays nothing out and scrolls nothing.
  area.scrollTo = () => {}
  Object.defineProperty(area, 'clientWidth', { value: width, configurable: true })
  Object.defineProperty(area, 'clientHeight', { value: height, configurable: true })
  ;(wrapper.vm as unknown as { measure: () => void }).measure()
  await wrapper.vm.$nextTick()
}

/** The widths the reader has emitted in `measure` events, in emission order. */
const getWidths = (wrapper: ReturnType<typeof mount>) =>
  (wrapper.emitted('measure') ?? []).map((one) => (one as [number])[0])

describe('the room a document is read in', () => {
  it('asks for a width once it knows how big the room is', async () => {
    const target = await reader(1000, 800)

    expect(getWidths(target)).toHaveLength(1)
    expect(getWidths(target)[0]).toBeGreaterThan(0)
  })

  it('asks for nothing while it has no room', async () => {
    // A document opening into a pane that has not been laid out yet has no
    // width to ask at, and a width nothing will draw at is a page drawn and
    // thrown away.
    const target = await reader(0, 0)

    expect(getWidths(target)).toHaveLength(0)
  })

  it('keeps the room it had when it is put out of sight', async () => {
    // Every tab of a pane is mounted while hidden, and a hidden tab has size
    // zero. Laying the row out at zero size requests every page again at a
    // width that is never drawn, and requests them a second time when the tab
    // becomes visible.
    const target = await reader(1000, 800)
    const before = target.find('.reader__row').attributes('style')

    await room(target, 0, 0)

    expect(getWidths(target)).toHaveLength(1)
    expect(target.find('.reader__row').attributes('style')).toBe(before)
  })

  it('asks again when the room really changes', async () => {
    // The first width is requested immediately: an opening document has nothing
    // drawn yet. Every later width is requested after the pane edge stops
    // moving.
    vi.useFakeTimers()
    try {
      const target = await reader(1000, 800)

      await room(target, 1400, 1100)
      await vi.advanceTimersByTimeAsync(300)

      expect(getWidths(target).length).toBeGreaterThan(1)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('the pages drawn', () => {
  it('draws the pages in the room and not the whole document', async () => {
    const target = await reader(1000, 800)

    const renderedPages = target.findAll('.reader__page')
    expect(renderedPages.length).toBeGreaterThan(0)
    expect(renderedPages.length).toBeLessThan(PAGES.length)
  })

  it('draws nothing at all for a document with no pages', async () => {
    const wrapper = mount(Reader, { props: { pages: [] }, attachTo: document.body })

    expect(wrapper.findAll('.reader__page')).toHaveLength(0)
    expect(wrapper.find('.reader__row').exists()).toBe(false)
  })
})

describe('the page it says it relies on', () => {
  /** Scrolls the row directly and dispatches the scroll event to the reader. */
  const scrollTo = async (wrapper: ReturnType<typeof mount>, to: number) => {
    const area = wrapper.find('.reader__viewport').element as HTMLElement
    area.scrollLeft = to
    await wrapper.find('.reader__viewport').trigger('scroll')
  }

  /** The pages the reader has requested through `go` events, in emission order. */
  const getTurns = (wrapper: ReturnType<typeof mount>) =>
    (wrapper.emitted('go') ?? []).map((one) => (one as [number])[0])

  it('says nothing while the row travels to the page it was turned to', async () => {
    // A turn is a scroll the browser animates, and the pages the row passes on
    // the way were not requested. Reporting one of them would scroll the row
    // back to it and undo the turn.
    const target = await reader(1000, 800)
    await target.setProps({ at: 4 })

    await scrollTo(target, 0)
    await scrollTo(target, 200)

    expect(getTurns(target)).toHaveLength(0)
  })

  it('says where the row remains once the hand has it', async () => {
    const target = await reader(1000, 800)

    await scrollTo(target, 4000)

    expect(getTurns(target).length).toBeGreaterThan(0)
    expect(getTurns(target).at(-1)).toBeGreaterThan(0)
  })

  it('turns again when the page changes before the row has returned it arrived', async () => {
    // The container size changes first and the resize event follows. A turn
    // requested in between targets a page the row is not on.
    const target = await reader(1000, 800)
    const area = target.find('.reader__viewport').element as HTMLElement
    const dispatched: number[] = []
    area.scrollTo = ((to: ScrollToOptions) => {
      dispatched.push(to.left ?? 0)
      area.scrollLeft = to.left ?? 0
    }) as typeof area.scrollTo

    await target.setProps({ at: 4 })
    await target.setProps({ at: 0 })

    expect(dispatched).toHaveLength(2)
    expect(dispatched[0]).toBeGreaterThan(0)
    expect(dispatched[1]).toBe(0)
  })
})

/** The handle the reader exposes to the host of the page it depends on. */
const getReaderHandle = (wrapper: ReturnType<typeof mount>) =>
  wrapper.vm as unknown as { handleKey: (event: KeyboardEvent) => boolean }

/** The pages the reader has requested through `go` events, in emission order. */
const getTurns = (wrapper: ReturnType<typeof mount>) =>
  (wrapper.emitted('go') ?? []).map((one) => (one as [number])[0])

const press = (key: string) => new KeyboardEvent('keydown', { key })

describe('a key struck over the pages', () => {
  it('asks for the page after this one, and the one before it', async () => {
    const wrapper = mount(Reader, { props: { pages: PAGES, at: 3 } })

    expect(getReaderHandle(wrapper).handleKey(press('ArrowRight'))).toBe(true)
    expect(getReaderHandle(wrapper).handleKey(press('ArrowLeft'))).toBe(true)

    expect(getTurns(wrapper)).toStrictEqual([4, 2])
    await wrapper.vm.$nextTick()
  })

  it('asks for either end of the document', () => {
    const wrapper = mount(Reader, { props: { pages: PAGES, at: 3 } })

    getReaderHandle(wrapper).handleKey(press('Home'))
    getReaderHandle(wrapper).handleKey(press('End'))

    expect(getTurns(wrapper)).toStrictEqual([0, PAGES.length - 1])
  })

  it('takes no key that turns no page, and leaves it to whoever is behind', () => {
    const wrapper = mount(Reader, { props: { pages: PAGES, at: 3 } })

    expect(getReaderHandle(wrapper).handleKey(press('k'))).toBe(false)
    expect(getReaderHandle(wrapper).handleKey(press('Enter'))).toBe(false)
    expect(getTurns(wrapper)).toStrictEqual([])
  })

  it('turns past neither end, so the key is left to whoever is behind', () => {
    const first = mount(Reader, { props: { pages: PAGES, at: 0 } })
    const last = mount(Reader, { props: { pages: PAGES, at: PAGES.length - 1 } })

    expect(getReaderHandle(first).handleKey(press('ArrowLeft'))).toBe(false)
    expect(getReaderHandle(last).handleKey(press('ArrowRight'))).toBe(false)
    expect(getTurns(first)).toStrictEqual([])
    expect(getTurns(last)).toStrictEqual([])
  })

  it('turns nothing where no page has come', () => {
    const wrapper = mount(Reader, { props: { pages: [] } })

    expect(getReaderHandle(wrapper).handleKey(press('ArrowRight'))).toBe(false)
  })
})

describe('where the keyboard remains', () => {
  it('is put on the pages, which are the region named for them', () => {
    const wrapper = mount(Reader, { props: { pages: PAGES }, attachTo: document.body })

    ;(wrapper.vm as unknown as { focusPages: () => void }).focusPages()

    const region = wrapper.get('[role="region"]').element
    expect(document.activeElement).toBe(region)
    expect(region.getAttribute('aria-label')).toBe(READER_WORDS.pages)
    wrapper.unmount()
  })
})
