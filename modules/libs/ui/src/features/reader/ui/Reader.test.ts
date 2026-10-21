import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Reader from './Reader.vue'
import { READER_WORDS } from '../lib/strip'

const PAGES = Array.from({ length: 8 }, () => ({ width: 612, height: 792 }))

/**
 * A reader in a room of a given size. Nothing in a test has a layout, so the
 * room says how big it is and the reader is told to take it again.
 */
const reader = async (wide: number, high: number) => {
  const wrapper = mount(Reader, {
    props: { pages: PAGES, picture: (page: number) => `/p/${page}` },
    attachTo: document.body,
  })
  await room(wrapper, wide, high)
  return wrapper
}

const room = async (wrapper: ReturnType<typeof mount>, wide: number, high: number) => {
  const area = wrapper.find('.reader__viewport').element as HTMLElement
  // jsdom lays nothing out and scrolls nothing.
  area.scrollTo = () => {}
  Object.defineProperty(area, 'clientWidth', { value: wide, configurable: true })
  Object.defineProperty(area, 'clientHeight', { value: high, configurable: true })
  ;(wrapper.vm as unknown as { measure: () => void }).measure()
  await wrapper.vm.$nextTick()
}

/** The widths the reader has asked for, in the order it asked. */
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
    // Every tab of a pane is mounted while it is hidden, and one out of sight
    // has no room. Laying the row out on nothing asks for every page again at a
    // width nothing will draw at, and asks for them all a second time when the
    // tab comes back.
    const target = await reader(1000, 800)
    const before = target.find('.reader__row').attributes('style')

    await room(target, 0, 0)

    expect(getWidths(target)).toHaveLength(1)
    expect(target.find('.reader__row').attributes('style')).toBe(before)
  })

  it('asks again when the room really changes', async () => {
    // The first width is asked for at once — a document opening has nothing
    // drawn and nothing to wait for. Every width after it waits for the edge of
    // the pane to stop moving.
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

    const rendered = target.findAll('.reader__page')
    expect(rendered.length).toBeGreaterThan(0)
    expect(rendered.length).toBeLessThan(PAGES.length)
  })

  it('draws nothing at all for a document with no pages', async () => {
    const wrapper = mount(Reader, { props: { pages: [] }, attachTo: document.body })

    expect(wrapper.findAll('.reader__page')).toHaveLength(0)
    expect(wrapper.find('.reader__row').exists()).toBe(false)
  })
})

describe('the page it says it relies on', () => {
  /** The row scrolled by hand, and the reader told about it. */
  const scrollTo = async (wrapper: ReturnType<typeof mount>, to: number) => {
    const area = wrapper.find('.reader__viewport').element as HTMLElement
    area.scrollLeft = to
    await wrapper.find('.reader__viewport').trigger('scroll')
  }

  /** Which pages the reader has asked to be turned to, in the order it asked. */
  const getTurns = (wrapper: ReturnType<typeof mount>) =>
    (wrapper.emitted('go') ?? []).map((one) => (one as [number])[0])

  it('says nothing while the row travels to the page it was turned to', async () => {
    // A turn is a scroll the browser animates, and the pages the row passes
    // over on the way are pages nobody turned to. Reporting one of them is
    // answered with a scroll back to it, and the turn is undone.
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
    // The room moves when it is told to and the event saying so comes after.
    // A turn asked for in between is a turn to a page the row is not on.
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

/** What the reader exposes to whoever holds the page it relies on. */
const getReaderHandle = (wrapper: ReturnType<typeof mount>) =>
  wrapper.vm as unknown as { handleKey: (event: KeyboardEvent) => boolean }

/** The pages the reader has asked to be turned to, in the order it asked. */
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
