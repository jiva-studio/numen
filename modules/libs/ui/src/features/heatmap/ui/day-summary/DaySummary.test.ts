/**
 * The account of one day of the grid: what was answered, how it was answered,
 * and how much of what was being reviewed came back.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DaySummary from './DaySummary.vue'
import type { Day } from '../../lib/heatmap'
import type { Words } from '../../lib/words'

const WORDS: Words = {
  names: (day) => `on ${day}`,
  answered: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
  recalled: 'came back',
}

const day = (held: Partial<Day> = {}): Day => ({
  day: '2026-03-11',
  did: 0,
  weight: 0,
  today: false,
  ahead: false,
  answered: 0,
  again: 0,
  hard: 0,
  good: 0,
  easy: 0,
  asked: 0,
  recalled: 0,
  ...held,
})

const account = (held: Partial<Day> = {}) =>
  mount(DaySummary, { props: { day: day(held), words: WORDS } })

/** Each of the four that was said, and how many of it. */
const four = (drawn: ReturnType<typeof account>) =>
  drawn.findAll('[data-day-summary="four"] li').map((one) => [
    one.attributes('data-tone'),
    one.get('[data-day-summary="said"]').text(),
    one.get('[data-day-summary="how-many"]').text(),
  ])

describe('a day that has been answered on', () => {
  it('names the day in the window’s own words', () => {
    expect(account({ did: 3 }).get('[data-day-summary="day"]').text()).toBe('on 2026-03-11')
  })

  it('counts what was answered', () => {
    expect(account({ did: 26 }).get('[data-day-summary="count"]').text()).toBe('26 answered')
  })

  it('lists the four in the order they are pressed, and leaves out the ones nobody pressed', () => {
    expect(four(account({ did: 24, again: 2, hard: 0, good: 18, easy: 4 }))).toEqual([
      ['again', 'Again', '2'],
      ['good', 'Good', '18'],
      ['easy', 'Easy', '4'],
    ])
  })

  it('says nothing of the four where none of them was pressed', () => {
    expect(account({ did: 5 }).find('[data-day-summary="four"]').exists()).toBe(false)
  })
})

describe('the share that came back', () => {
  it('is what came back of what was asked, to the nearest whole', () => {
    expect(
      account({ did: 26, asked: 24, recalled: 20 }).get('[data-day-summary="came"]').text(),
    ).toBe('83% came back')
  })

  // Nothing the person is already reviewing was asked, so there is no share to
  // give and none is invented.
  it('is not given where nothing being reviewed was asked', () => {
    expect(
      account({ did: 12, asked: 0, recalled: 0 }).find('[data-day-summary="came"]').exists(),
    ).toBe(false)
  })

  it('is the whole of it where everything asked came back', () => {
    expect(account({ did: 9, asked: 9, recalled: 9 }).get('[data-day-summary="came"]').text()).toBe(
      '100% came back',
    )
  })
})

describe('a day nobody answered on', () => {
  it('says so, and says nothing else', () => {
    const drawn = account()
    expect(drawn.get('[data-day-summary="count"]').text()).toBe('Nothing answered')
    expect(drawn.find('[data-day-summary="four"]').exists()).toBe(false)
    expect(drawn.find('[data-day-summary="came"]').exists()).toBe(false)
  })
})

describe('a day still to come', () => {
  it('says what falls on it rather than what was done on it', () => {
    expect(account({ ahead: true, did: 7 }).get('[data-day-summary="count"]').text()).toBe('7 to come')
  })

  // A day ahead was not answered on, so the four and the share are about
  // nothing.
  it('says none of the four and no share, whatever it carries', () => {
    const drawn = account({ ahead: true, did: 7, again: 2, asked: 4, recalled: 3 })
    expect(drawn.find('[data-day-summary="four"]').exists()).toBe(false)
    expect(drawn.find('[data-day-summary="came"]').exists()).toBe(false)
  })

  it('says nothing falls on a day nothing falls on', () => {
    expect(account({ ahead: true }).get('[data-day-summary="count"]').text()).toBe('Nothing answered')
  })
})
