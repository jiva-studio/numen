import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Reader from './Reader.vue'

const SHEETS = Array.from({ length: 8 }, () => ({ wide: 612, high: 792 }))

/**
 * A reader in a room of a given size. Nothing in a test has a layout, so the
 * room says how big it is and the reader is told to take it again.
 */
const reader = async (wide: number, high: number) => {
  const held = mount(Reader, {
    props: { pages: SHEETS.length, sheets: SHEETS, picture: (page: number) => `/p/${page}` },
    attachTo: document.body,
  })
  await room(held, wide, high)
  return held
}

const room = async (held: ReturnType<typeof mount>, wide: number, high: number) => {
  const area = held.find('.reader__room').element as HTMLElement
  // jsdom lays nothing out and scrolls nothing.
  area.scrollTo = () => {}
  Object.defineProperty(area, 'clientWidth', { value: wide, configurable: true })
  Object.defineProperty(area, 'clientHeight', { value: high, configurable: true })
  ;(held.vm as unknown as { measure: () => void }).measure()
  await held.vm.$nextTick()
}

/** The widths the reader has asked for, in the order it asked. */
const asked = (held: ReturnType<typeof mount>) =>
  (held.emitted('wide') ?? []).map((one) => (one as [number])[0])

describe('the room a document is read in', () => {
  it('asks for a width once it knows how big the room is', async () => {
    const held = await reader(1000, 800)

    expect(asked(held)).toHaveLength(1)
    expect(asked(held)[0]).toBeGreaterThan(0)
  })

  it('asks for nothing while it has no room', async () => {
    // A document opening into a pane that has not been laid out yet has no
    // width to ask at, and a width nothing will draw at is a page drawn and
    // thrown away.
    const held = await reader(0, 0)

    expect(asked(held)).toHaveLength(0)
  })

  it('keeps the room it had when it is put out of sight', async () => {
    // Every tab of a pane is mounted while it is hidden, and one out of sight
    // has no room. Laying the row out on nothing asks for every page again at a
    // width nothing will draw at, and asks for them all a second time when the
    // tab comes back.
    const held = await reader(1000, 800)
    const before = held.find('.reader__row').attributes('style')

    await room(held, 0, 0)

    expect(asked(held)).toHaveLength(1)
    expect(held.find('.reader__row').attributes('style')).toBe(before)
  })

  it('asks again when the room really changes', async () => {
    // The first width is asked for at once — a document opening has nothing
    // drawn and nothing to wait for. Every width after it waits for the edge of
    // the pane to stop moving.
    vi.useFakeTimers()
    try {
      const held = await reader(1000, 800)

      await room(held, 1400, 1100)
      await vi.advanceTimersByTimeAsync(300)

      expect(asked(held).length).toBeGreaterThan(1)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('the pages drawn', () => {
  it('draws the pages in the room and not the whole document', async () => {
    const held = await reader(1000, 800)

    const drawn = held.findAll('.reader__page')
    expect(drawn.length).toBeGreaterThan(0)
    expect(drawn.length).toBeLessThan(SHEETS.length)
  })

  it('draws nothing at all for a document with no pages', async () => {
    const held = mount(Reader, { props: { pages: 0, sheets: [] }, attachTo: document.body })

    expect(held.findAll('.reader__page')).toHaveLength(0)
    expect(held.find('.reader__row').exists()).toBe(false)
  })
})
