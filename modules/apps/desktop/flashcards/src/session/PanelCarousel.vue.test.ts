// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import PanelCarousel from './PanelCarousel.vue'
import type { PanelPlace } from './PanelCarousel.vue'

/**
 * The three on the screen. jsdom lays nothing out, so the strip is given the
 * widths it would have had: a panel either side of a card the width of the
 * window, with a space between each pair.
 */
const strip = (at: PanelPlace = 'here') => {
  const one = mount(PanelCarousel, {
    // The window is listening and answers by setting the prop, so what is in the
    // window stays the window's answer. A strip nobody listens to keeps its own.
    props: { at, 'onUpdate:at': () => {} },
    slots: {
      before: '<p>the reading</p>',
      default: '<p>the card</p>',
      after: '<p>the chat</p>',
    },
  })
  const window_ = one.find('.carousel').element as HTMLElement
  Object.defineProperty(window_, 'scrollWidth', { value: 1040, configurable: true })
  Object.defineProperty(window_, 'clientWidth', { value: 600, configurable: true })
  Object.defineProperty(one.find('.carousel__here').element, 'offsetLeft', {
    value: 220,
    configurable: true,
  })
  window_.setPointerCapture = () => {}
  // The widths are only known now, so the strip is stood on its stop the way a
  // resize stands it on one.
  window.dispatchEvent(new Event('resize'))
  return { one, window_ }
}

/** A hand going down on the strip, moving, and coming off it. */
const hand = (kind: string, clientX: number) =>
  new MouseEvent(kind, { button: 0, clientX, bubbles: true })

/** The strip left where a hand or a wheel put it, and the window told. */
const ran = async (window_: HTMLElement, to: number) => {
  window_.scrollLeft = to
  window_.dispatchEvent(new Event('scroll'))
  await nextTick()
}

describe('a card with a panel on either side of it', () => {
  it('draws all three, wherever the strip is standing', () => {
    for (const at of ['before', 'here', 'after'] as const) {
      const { one } = strip(at)
      expect(one.text()).toContain('the reading')
      expect(one.text()).toContain('the card')
      expect(one.text()).toContain('the chat')
    }
  })

  // The window opens on the card, and it is put there rather than taken there:
  // a card seen sliding into place is a card that looks like it is leaving.
  it('rests on the card in the middle', () => {
    expect(strip().window_.scrollLeft).toBe(220)
  })

  // Whichever is out of the window is reached by nothing: not the keyboard, and
  // not what reads the screen aloud.
  it('puts whichever is out of the window beyond reach', () => {
    const reading = strip('before').one
    expect(reading.find('.carousel__before').attributes('inert')).toBeUndefined()
    expect(reading.find('.carousel__here').attributes('inert')).toBeDefined()
    expect(reading.find('.carousel__after').attributes('inert')).toBeDefined()

    const card = strip('here').one
    expect(card.find('.carousel__before').attributes('inert')).toBeDefined()
    expect(card.find('.carousel__here').attributes('inert')).toBeUndefined()
    expect(card.find('.carousel__after').attributes('inert')).toBeDefined()

    const chat = strip('after').one
    expect(chat.find('.carousel__before').attributes('inert')).toBeDefined()
    expect(chat.find('.carousel__here').attributes('inert')).toBeDefined()
    expect(chat.find('.carousel__after').attributes('inert')).toBeUndefined()
  })

  // Asked for by a key rather than a hand, the strip is taken there rather than
  // put there, so it reads as the same movement either way.
  it('scrolls to whichever of the three is asked for', async () => {
    const { one, window_ } = strip()

    await one.setProps({ at: 'after' })
    expect(window_.scrollLeft).toBe(440)

    await one.setProps({ at: 'before' })
    expect(window_.scrollLeft).toBe(0)

    await one.setProps({ at: 'here' })
    expect(window_.scrollLeft).toBe(220)
  })

  // The stops are the layout's own: where the card stands, and how far the
  // strip goes. The space the three stand apart by is in them already.
  it('stops where the card stands and not at a width worked out', async () => {
    const { one, window_ } = strip('before')
    Object.defineProperty(one.find('.carousel__here').element, 'offsetLeft', {
      value: 300,
      configurable: true,
    })

    await one.setProps({ at: 'here' })
    expect(window_.scrollLeft).toBe(300)
  })
})

describe('a hand on the strip', () => {
  it('asks for whichever it has taken the strip nearest to', async () => {
    const chat = strip()
    await ran(chat.window_, 400)
    expect(chat.one.emitted('update:at')).toEqual([['after']])

    const reading = strip()
    await ran(reading.window_, 20)
    expect(reading.one.emitted('update:at')).toEqual([['before']])

    const card = strip('before')
    await ran(card.window_, 190)
    expect(card.one.emitted('update:at')).toEqual([['here']])
  })

  it('says nothing while the strip is still nearest where it stood', async () => {
    const { one, window_ } = strip()
    await ran(window_, 180)
    expect(one.emitted('update:at')).toBeUndefined()
  })

  // A strip taken somewhere by a key passes the others on the way, and that is
  // not a person asking for one of them. It is what escape closes the panel by.
  it('says nothing about where it is passing through on its way', async () => {
    const { one, window_ } = strip('after')

    await one.setProps({ at: 'here' })
    await ran(window_, 400)

    expect(one.emitted('update:at')).toBeUndefined()
  })

  // A hand lets go where it likes, and the strip settles on the nearest stop
  // rather than staying between two of them.
  it('settles on the nearest stop when it is let go', async () => {
    const { one, window_ } = strip()

    window_.dispatchEvent(hand('pointerdown', 500))
    window_.dispatchEvent(hand('pointermove', 380))
    await nextTick()
    expect(window_.scrollLeft).toBe(340)

    window_.dispatchEvent(hand('pointerup', 380))
    await nextTick()
    // The window took the hand's word for it, as a window that opened the panel
    // does.
    await one.setProps({ at: 'after' })
    await nextTick()

    expect(window_.scrollLeft).toBe(440)
    expect(one.emitted('update:at')).toEqual([['after']])
  })

  // A panel the window refused to open is one the strip must not be left
  // standing on: what is in the window is the window's answer, not the strip's.
  it('goes back off a panel the window did not open', async () => {
    const { one, window_ } = strip()

    window_.dispatchEvent(hand('pointerdown', 500))
    window_.dispatchEvent(hand('pointermove', 380))
    window_.dispatchEvent(hand('pointerup', 380))
    await nextTick()
    await nextTick()

    expect(one.emitted('update:at')).toEqual([['after']])
    expect(window_.scrollLeft).toBe(220)
  })

  it('settles back on where it started when it is let go short of the next', async () => {
    const { one, window_ } = strip()

    window_.dispatchEvent(hand('pointerdown', 500))
    window_.dispatchEvent(hand('pointermove', 440))
    window_.dispatchEvent(hand('pointerup', 440))
    await nextTick()
    await nextTick()

    expect(window_.scrollLeft).toBe(220)
    expect(one.emitted('update:at')).toBeUndefined()
  })
})

// A wheel and a trackpad leave the strip wherever they ran out, and nothing
// comes off them to say the gesture is over. What settles it is a wait.
describe('a wheel on the strip', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  const waits = async (ms: number) => {
    vi.advanceTimersByTime(ms)
    await nextTick()
    await nextTick()
  }

  it('is taken the rest of the way once it has stopped', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await ran(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])
    await one.setProps({ at: 'after' })

    await waits(200)
    expect(window_.scrollLeft).toBe(440)
  })

  it('is settled even where it stopped short of the next', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await ran(window_, 260)
    expect(one.emitted('update:at')).toBeUndefined()

    await waits(200)
    expect(window_.scrollLeft).toBe(220)
  })

  // A panel the window refused to open is one the strip must not be left
  // standing on, and a wheel is settled the same way a hand is.
  it('is taken back off a stop the window did not take', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await ran(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])

    await waits(200)
    expect(window_.scrollLeft).toBe(220)
  })

  // A press that moved nothing sends the strip nowhere. A move that moves
  // nothing never arrives, and would leave the strip deaf until its time was up.
  it('is heard after a press that moved the strip nowhere', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    window_.dispatchEvent(hand('pointerdown', 500))
    window_.dispatchEvent(hand('pointerup', 500))
    await nextTick()
    await nextTick()

    await ran(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])
  })

  // A hand that turns the wheel during a move would otherwise leave the strip
  // standing between two of the three, with nothing left to pull it to either.
  it('is settled when it is turned while the strip is being taken somewhere', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await one.setProps({ at: 'after' })
    await ran(window_, 300)

    await waits(200)
    expect(window_.scrollLeft).toBe(440)
  })
})
