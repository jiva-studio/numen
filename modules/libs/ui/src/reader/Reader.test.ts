import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Reader from './Reader.vue'

const PAGES = Array.from({ length: 8 }, () => ({ width: 612, height: 792 }))

/**
 * A reader in a room of a given size. Nothing in a test has a layout, so the
 * room says how big it is and the reader is told to take it again.
 */
const reader = async (wide: number, high: number) => {
  const held = mount(Reader, {
    props: { pages: PAGES, picture: (page: number) => `/p/${page}` },
    attachTo: document.body,
  })
  await room(held, wide, high)
  return held
}

const room = async (held: ReturnType<typeof mount>, wide: number, high: number) => {
  const area = held.find('.reader__viewport').element as HTMLElement
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
    expect(drawn.length).toBeLessThan(PAGES.length)
  })

  it('draws nothing at all for a document with no pages', async () => {
    const held = mount(Reader, { props: { pages: [] }, attachTo: document.body })

    expect(held.findAll('.reader__page')).toHaveLength(0)
    expect(held.find('.reader__row').exists()).toBe(false)
  })
})

describe('the page it says it stands on', () => {
  /** The row scrolled by hand, and the reader told about it. */
  const moved = async (held: ReturnType<typeof mount>, to: number) => {
    const area = held.find('.reader__viewport').element as HTMLElement
    area.scrollLeft = to
    await held.find('.reader__viewport').trigger('scroll')
  }

  /** Which pages the reader has asked to be turned to, in the order it asked. */
  const turned = (held: ReturnType<typeof mount>) =>
    (held.emitted('go') ?? []).map((one) => (one as [number])[0])

  it('says nothing while the row travels to the page it was turned to', async () => {
    // A turn is a scroll the browser animates, and the pages the row passes
    // over on the way are pages nobody turned to. Reporting one of them is
    // answered with a scroll back to it, and the turn is undone.
    const held = await reader(1000, 800)
    await held.setProps({ at: 4 })

    await moved(held, 0)
    await moved(held, 200)

    expect(turned(held)).toHaveLength(0)
  })

  it('says where the row stands once the hand has it', async () => {
    const held = await reader(1000, 800)

    await moved(held, 4000)

    expect(turned(held).length).toBeGreaterThan(0)
    expect(turned(held).at(-1)).toBeGreaterThan(0)
  })

  it('turns again when the page changes before the row has said it arrived', async () => {
    // The room moves when it is told to and the event saying so comes after.
    // A turn asked for in between is a turn to a page the row is not on.
    const held = await reader(1000, 800)
    const area = held.find('.reader__viewport').element as HTMLElement
    const sent: number[] = []
    area.scrollTo = ((to: ScrollToOptions) => {
      sent.push(to.left ?? 0)
      area.scrollLeft = to.left ?? 0
    }) as typeof area.scrollTo

    await held.setProps({ at: 4 })
    await held.setProps({ at: 0 })

    expect(sent).toHaveLength(2)
    expect(sent[0]).toBeGreaterThan(0)
    expect(sent[1]).toBe(0)
  })
})
