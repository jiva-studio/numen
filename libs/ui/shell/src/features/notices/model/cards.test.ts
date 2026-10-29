/**
 * Which cards stand, when the clock is kept running for them, and where the
 * keyboard goes when one is put away.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, h, nextTick, shallowRef } from 'vue'
import { useNoticeCards, type NoticeCardsState } from './cards'
import { MIN_DURATION } from '../lib/duration'
import type { Notice } from '../lib/notice'
import { getRemainingWord } from '../lib/tally'

const work = (id: string): Notice => ({ id, text: `Indexing ${id}`, isWorking: true })
const request = (id: string): Notice => ({
  id,
  text: `Filed ${id}`,
  stay: 'kept',
  isUserStarted: true,
})
const report = (id: string, text = `Renamed ${id}`): Notice => ({
  id,
  text,
  stay: 'read',
  isUserStarted: true,
})
const countMetric = (id: string, isDone: number): Notice => ({
  id,
  text: `Learning ${id}`,
  isWorking: true,
  isUserStarted: true,
  done: isDone,
  total: 100,
  counting: 'items',
})

let mounted: VueWrapper | undefined

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
})

afterEach(() => {
  mounted?.unmount()
  mounted = undefined
  vi.useRealTimers()
  document.body.innerHTML = ''
})

/** The cards over a list of notices and a clock, both wound by the test. */
function stand(initial: readonly Notice[], options: { room?: number; wait?: number } = {}) {
  const notices = shallowRef<readonly Notice[]>(initial)
  const stack = shallowRef<HTMLElement | null>(null)
  const clock = { at: 0 }
  const dismiss = vi.fn<(id: string) => void>()
  let state!: NoticeCardsState
  mounted = mount(
    defineComponent({
      setup() {
        state = useNoticeCards({
          getNotices: () => notices.value,
          stack,
          getWait: () => options.wait ?? 10_000,
          getFreeSpace: () => options.room ?? 4,
          getNow: () => clock.at,
          isHidden: () => false,
          dismiss,
        })
        return () => h('div')
      },
    }),
    { attachTo: document.body },
  )
  /** The clock forward by a duration, and one tick of the interval. */
  const wind = async (delta: number): Promise<void> => {
    clock.at += delta
    vi.advanceTimersByTime(250)
    await nextTick()
  }
  const ids = (): string[] => state.notices.value.map((one) => one.id)
  return { notices, state, dismiss, wind, ids }
}

/** A focusable stand-in for the way away on a card. */
function button(): HTMLButtonElement {
  const one = document.createElement('button')
  document.body.append(one)
  return one
}

describe('the clock behind the corner', () => {
  it('does not run while there is nothing to wait for', () => {
    stand([])

    expect(vi.getTimerCount()).toBe(0)
  })

  it('runs for work that has not yet lasted long enough to be drawn, and draws it once it has', async () => {
    const { wind, ids } = stand([work('a')])

    expect(vi.getTimerCount()).toBe(1)
    expect(ids()).toEqual([])

    await wind(10_000)

    expect(ids()).toEqual(['a'])
    expect(vi.getTimerCount()).toBe(0)
  })

  it('runs for the work still waiting when other cards are already drawn', async () => {
    const { wind, ids } = stand([request('k'), work('a')])

    expect(ids()).toEqual(['k'])
    expect(vi.getTimerCount()).toBe(1)

    await wind(10_000)

    expect(ids()).toEqual(['k', 'a'])
  })

  it('does not run for a card that is drawn and stays until it is put away', () => {
    stand([request('k')])

    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not run for work a person has put away', async () => {
    const { state } = stand([{ ...work('a'), isUserStarted: true }])
    expect(vi.getTimerCount()).toBe(0)

    await state.dismissManually('a')
    await nextTick()

    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not run for a report that has been read', async () => {
    const { wind } = stand([report('a')])
    expect(vi.getTimerCount()).toBe(1)

    await wind(MIN_DURATION * 10)

    expect(vi.getTimerCount()).toBe(0)
  })

  it('runs for a report on its way out, along with a card that stays', () => {
    stand([report('a'), request('k')])

    expect(vi.getTimerCount()).toBe(1)
  })

  it('does not run for a report that is not read in passing', () => {
    stand([report('a', Array.from({ length: 21 }, (_, i) => `w${i}`).join(' '))])

    expect(vi.getTimerCount()).toBe(0)
  })

  it('runs for a count, along with a card that does not count', () => {
    stand([request('k'), countMetric('c', 10)])

    expect(vi.getTimerCount()).toBe(1)
  })

  it('does not run for a card that is not counting', () => {
    stand([request('k'), request('j')])

    expect(vi.getTimerCount()).toBe(0)
  })

  it('stops when the work that needed it is over', async () => {
    const { notices } = stand([countMetric('c', 10)])
    expect(vi.getTimerCount()).toBe(1)

    notices.value = []
    await nextTick()

    expect(vi.getTimerCount()).toBe(0)
  })
})

describe('what is left on a count', () => {
  it('is nothing for a card that does not count', () => {
    const { state } = stand([request('k')])

    expect(state.leftOn(request('k'))).toBe('')
  })

  it('is nothing before the count has moved', () => {
    const { state } = stand([countMetric('c', 10)])

    expect(state.leftOn(countMetric('c', 10))).toBe('')
  })

  it('is nothing for a count the corner has not yet seen', () => {
    const { state } = stand([])

    expect(state.leftOn(countMetric('unseen', 10))).toBe('')
  })

  it('is what the rate it moved at leaves', async () => {
    const { state, notices, wind } = stand([countMetric('c', 0)])

    await wind(1_000)
    notices.value = [countMetric('c', 10)]
    await nextTick()

    expect(state.leftOn(countMetric('c', 10))).toBe(getRemainingWord(90, 10))
    expect(state.leftOn(countMetric('c', 10))).not.toBe('')
  })
})

describe('a card that has been read', () => {
  it('names its caller once, however often the corner is looked at again', async () => {
    const { dismiss, notices, wind } = stand([report('a')])

    await wind(MIN_DURATION * 10)
    notices.value = [report('a')]
    await wind(1_000)
    notices.value = [report('a')]
    await wind(1_000)

    expect(dismiss.mock.calls).toEqual([['a']])
  })

  it('names its caller again when the same card returns and is read again', async () => {
    const { dismiss, notices, wind } = stand([report('a')])
    await wind(MIN_DURATION * 10)

    notices.value = []
    await nextTick()
    notices.value = [report('a')]
    await nextTick()
    await wind(MIN_DURATION * 10)

    expect(dismiss.mock.calls).toEqual([['a'], ['a']])
  })

  it('is not forgotten while its caller still passes it in', async () => {
    const { dismiss, notices, wind } = stand([report('a'), request('k')])
    await wind(MIN_DURATION * 10)

    notices.value = [report('a'), request('k'), request('j')]
    await nextTick()
    notices.value = [report('a'), request('k')]
    await wind(1_000)

    expect(dismiss.mock.calls).toEqual([['a']])
  })
})

describe('the cards folded away behind the rest', () => {
  const three = [request('a'), request('b'), request('c')]

  it('stay open while they are still more than fit', async () => {
    const { state, notices } = stand(three, { room: 2 })
    state.isOpened.value = true
    await nextTick()

    notices.value = [...three, request('d')]
    await nextTick()

    expect(state.isOpened.value).toBe(true)
    expect(state.folds.value.visible).toHaveLength(4)
  })

  it('close once everything fits', async () => {
    const { state } = stand(three, { room: 2 })
    state.isOpened.value = true
    await nextTick()
    expect(state.folds.value.visible).toHaveLength(3)

    await state.dismissManually('a')
    await nextTick()

    expect(state.isOpened.value).toBe(false)
    expect(state.folds.value.overflowCount).toBe(0)
  })

  it('close when fewer than the room remain', async () => {
    const { state } = stand(three, { room: 4 })
    state.isOpened.value = true
    await nextTick()

    await state.dismissManually('a')
    await nextTick()

    expect(state.isOpened.value).toBe(false)
  })
})

describe('putting a card away by hand', () => {
  const three = [request('a'), request('b'), request('c')]

  /** Three cards standing, each held under its notice, with the keyboard on one. */
  function hold(focused: 'a' | 'b' | 'c' | undefined) {
    const status = stand(three)
    const ways = { a: button(), b: button(), c: button() }
    for (const id of ['a', 'b', 'c'] as const) status.state.holdCard(id, { way: ways[id] })
    if (focused) ways[focused].focus()
    return { ...status, ways }
  }

  it('names its caller and removes the card', async () => {
    const { state, dismiss, ids } = hold(undefined)

    await state.dismissManually('b')

    expect(dismiss.mock.calls).toEqual([['b']])
    expect(ids()).toEqual(['a', 'c'])
  })

  it('does not name its caller a second time when the card is read out afterwards', async () => {
    const { state, dismiss, notices } = stand([report('a')])

    await state.dismissManually('a')
    notices.value = [report('a')]
    await nextTick()

    expect(dismiss.mock.calls).toEqual([['a']])
  })

  it('leaves the keyboard on the card that takes the place of the one that went', async () => {
    const { state, ways } = hold('b')

    await state.dismissManually('b')

    expect(document.activeElement).toBe(ways.c)
  })

  it('leaves the keyboard on the first card when the first went', async () => {
    const { state, ways } = hold('a')

    await state.dismissManually('a')

    expect(document.activeElement).toBe(ways.b)
  })

  it('leaves the keyboard on the last card when the last went', async () => {
    const { state, ways } = hold('c')

    await state.dismissManually('c')

    expect(document.activeElement).toBe(ways.b)
  })

  it('does not move the keyboard when it was not on the card', async () => {
    const { state, ways } = hold('a')

    await state.dismissManually('c')

    expect(document.activeElement).toBe(ways.a)
  })

  it('does not move the keyboard when it was somewhere else entirely', async () => {
    const { state } = hold(undefined)
    const elsewhere = button()
    elsewhere.focus()

    await state.dismissManually('b')

    expect(document.activeElement).toBe(elsewhere)
  })

  it('does not move the keyboard to a card that has been taken down', async () => {
    const { state, ways } = hold('b')
    state.holdCard('b', null)

    await state.dismissManually('b')

    expect(document.activeElement).toBe(ways.b)
  })

  it('leaves the keyboard alone when it was the only card', async () => {
    const status = stand([request('a')])
    const way = button()
    status.state.holdCard('a', { way })
    way.focus()

    await expect(status.state.dismissManually('a')).resolves.toBeUndefined()

    expect(status.dismiss.mock.calls).toEqual([['a']])
  })

  it('puts away a card that was never retained', async () => {
    const { state, dismiss } = stand(three)

    await expect(state.dismissManually('b')).resolves.toBeUndefined()

    expect(dismiss.mock.calls).toEqual([['b']])
  })

  it('moves on when the card that takes its place has no way away', async () => {
    const status = stand(three)
    const way = button()
    status.state.holdCard('b', { way })
    status.state.holdCard('c', { way: null })
    way.focus()

    await expect(status.state.dismissManually('b')).resolves.toBeUndefined()
  })

  it('moves on when the card that takes its place is not retained', async () => {
    const status = stand(three)
    const way = button()
    status.state.holdCard('b', { way })
    way.focus()

    await expect(status.state.dismissManually('b')).resolves.toBeUndefined()
  })
})
