// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import PanelCarousel from './PanelCarousel.vue'
import type { PanelPlace } from '../model/carousel'

/**
 * The three panels. jsdom performs no layout, so the strip is given fixed widths:
 * a panel on each side of a card as wide as the container, with
 * a gap between each pair.
 */
const strip = (place: PanelPlace = 'here') => {
  const one = mount(PanelCarousel, {
    // The parent handles the event by setting the prop, so the parent owns
    // the position. A strip without a listener keeps its own position.
    props: { at: place, 'onUpdate:at': () => undefined },
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
  window_.setPointerCapture = () => undefined
  // The widths are only known now, so the strip is snapped to its stop the way a
  // resize snaps it.
  window.dispatchEvent(new Event('resize'))
  return { one, window_ }
}

/** A pointer event on the strip: press, move or release. */
const hand = (kind: string, clientX: number) =>
  new MouseEvent(kind, { button: 0, clientX, bubbles: true })

/** Scrolls the strip to a position and notifies the container. */
const scrollStrip = async (window_: HTMLElement, to: number) => {
  window_.scrollLeft = to
  window_.dispatchEvent(new Event('scroll'))
  await nextTick()
}

describe('a card with a panel on either side of it', () => {
  it('draws all three, wherever the strip is standing', () => {
    for (const place of ['before', 'here', 'after'] as const) {
      const { one } = strip(place)
      expect(one.text()).toContain('the reading')
      expect(one.text()).toContain('the card')
      expect(one.text()).toContain('the chat')
    }
  })

  // The window opens on the card, and it is put there and not taken there:
  // a card seen sliding into place is a card that looks like it is leaving.
  it('rests on the card in the middle', () => {
    expect(strip().window_.scrollLeft).toBe(220)
  })

  // Whichever is out of the window is reached by nothing: not the keyboard, and
  // not what reads the screen aloud.
  it('puts whichever is out of the window beyond reach', () => {
    const readText = strip('before').one
    expect(readText.find('.carousel__before').attributes('inert')).toBeUndefined()
    expect(readText.find('.carousel__here').attributes('inert')).toBeDefined()
    expect(readText.find('.carousel__after').attributes('inert')).toBeDefined()

    const card = strip('here').one
    expect(card.find('.carousel__before').attributes('inert')).toBeDefined()
    expect(card.find('.carousel__here').attributes('inert')).toBeUndefined()
    expect(card.find('.carousel__after').attributes('inert')).toBeDefined()

    const chat = strip('after').one
    expect(chat.find('.carousel__before').attributes('inert')).toBeDefined()
    expect(chat.find('.carousel__here').attributes('inert')).toBeDefined()
    expect(chat.find('.carousel__after').attributes('inert')).toBeUndefined()
  })

  //   // When requested by a key and not a pointer, the strip scrolls to the stop
  //   // with the same movement as a pointer scroll.
  it('scrolls to whichever of the three is requested for', async () => {
    const { one, window_ } = strip()

    await one.setProps({ at: 'after' })
    expect(window_.scrollLeft).toBe(440)

    await one.setProps({ at: 'before' })
    expect(window_.scrollLeft).toBe(0)

    await one.setProps({ at: 'here' })
    expect(window_.scrollLeft).toBe(220)
  })

  //   // The stops come from the layout: the position of the card and the scroll
  //   // extent of the strip. The gaps between the three panels are already included.
  it('stops where the card remains and not at a width worked out', async () => {
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
    await scrollStrip(chat.window_, 400)
    expect(chat.one.emitted('update:at')).toEqual([['after']])

    const readText = strip()
    await scrollStrip(readText.window_, 20)
    expect(readText.one.emitted('update:at')).toEqual([['before']])

    const card = strip('before')
    await scrollStrip(card.window_, 190)
    expect(card.one.emitted('update:at')).toEqual([['here']])
  })

  it('says nothing while the strip is still nearest where it stood', async () => {
    const { one, window_ } = strip()
    await scrollStrip(window_, 180)
    expect(one.emitted('update:at')).toBeUndefined()
  })

  //   // A strip scrolled by a key passes the intermediate stops, and that is not a
  //   // request for one of them. Escape closes the panel through this path.
  it('says nothing about where it is passing through on its way', async () => {
    const { one, window_ } = strip('after')

    await one.setProps({ at: 'here' })
    await scrollStrip(window_, 400)

    expect(one.emitted('update:at')).toBeUndefined()
  })

  //   // A pointer release can land anywhere, and the strip settles on the nearest
  //   // stop and does not stay between two of them.
  it('settles on the nearest stop when it is let go', async () => {
    const { one, window_ } = strip()

    window_.dispatchEvent(hand('pointerdown', 500))
    window_.dispatchEvent(hand('pointermove', 380))
    await nextTick()
    expect(window_.scrollLeft).toBe(340)

    window_.dispatchEvent(hand('pointerup', 380))
    await nextTick()
    //     // The parent accepts the emitted position, as a parent that opened the panel
    //     // does.
    await one.setProps({ at: 'after' })
    await nextTick()

    expect(window_.scrollLeft).toBe(440)
    expect(one.emitted('update:at')).toEqual([['after']])
  })

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

  const wait = async (ms: number) => {
    vi.advanceTimersByTime(ms)
    await nextTick()
    await nextTick()
  }

  it('is taken the rest of the way once it has stopped', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await scrollStrip(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])
    await one.setProps({ at: 'after' })

    await wait(200)
    expect(window_.scrollLeft).toBe(440)
  })

  it('is settled even where it stopped short of the next', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await scrollStrip(window_, 260)
    expect(one.emitted('update:at')).toBeUndefined()

    await wait(200)
    expect(window_.scrollLeft).toBe(220)
  })

  //   // A scroll to a stop that the parent does not accept returns the strip to the
  //   // current stop, and a wheel scroll settles the same way as a pointer drag.
  it('is taken back off a stop the window did not take', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await scrollStrip(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])

    await wait(200)
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

    await scrollStrip(window_, 380)
    expect(one.emitted('update:at')).toEqual([['after']])
  })

  //   // A wheel scroll during a programmatic move leaves the strip
  //   // between two of the three stops with no snap to either.
  it('is settled when it is turned while the strip is being taken somewhere', async () => {
    vi.useFakeTimers()
    const { one, window_ } = strip()

    await one.setProps({ at: 'after' })
    await scrollStrip(window_, 300)

    await wait(200)
    expect(window_.scrollLeft).toBe(440)
  })
})
