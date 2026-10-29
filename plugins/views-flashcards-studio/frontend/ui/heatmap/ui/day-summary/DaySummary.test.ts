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
  reviewed: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
  recalled: 'came back',
}

const day = (overrides: Partial<Day> = {}): Day => ({
  day: '2026-03-11',
  did: 0,
  weight: 0,
  isToday: false,
  isAhead: false,
  reviewed: 0,
  again: 0,
  hard: 0,
  good: 0,
  easy: 0,
  tested: 0,
  recalled: 0,
  ...overrides,
})

const mountDetail = (overrides: Partial<Day> = {}) =>
  mount(DaySummary, { props: { day: day(overrides), words: WORDS } })

/** Each of the four ratings, with its count. */
const getRatings = (wrapper: ReturnType<typeof mountDetail>) =>
  wrapper
    .findAll('[data-day-detail="ratings"] li')
    .map((one) => [
      one.attributes('data-tone'),
      one.get('[data-day-detail="label"]').text(),
      one.get('[data-day-detail="rating-count"]').text(),
    ])

describe('a day that has been answered on', () => {
  it('names the day in the window’s own words', () => {
    expect(mountDetail({ did: 3 }).get('[data-day-detail="day"]').text()).toBe('on 2026-03-11')
  })

  it('counts what was answered', () => {
    expect(mountDetail({ did: 26 }).get('[data-day-detail="count"]').text()).toBe('26 answered')
  })

  it('lists the four in the order they are pressed, and leaves out the ones nobody pressed', () => {
    expect(getRatings(mountDetail({ did: 24, again: 2, hard: 0, good: 18, easy: 4 }))).toEqual([
      ['again', 'Again', '2'],
      ['good', 'Good', '18'],
      ['easy', 'Easy', '4'],
    ])
  })

  it('says nothing of the four where none of them was pressed', () => {
    expect(mountDetail({ did: 5 }).find('[data-day-detail="ratings"]').exists()).toBe(false)
  })
})

describe('the share that came back', () => {
  it('is what came back of what was requested, to the nearest whole', () => {
    expect(
      mountDetail({ did: 26, tested: 24, recalled: 20 })
        .get('[data-day-detail="recall-rate"]')
        .text(),
    ).toBe('83% came back')
  })

  // No cards under review were requested, so there is no share to report and
  // none is invented.
  it('is not given where nothing being reviewed was requested', () => {
    expect(
      mountDetail({ did: 12, tested: 0, recalled: 0 })
        .find('[data-day-detail="recall-rate"]')
        .exists(),
    ).toBe(false)
  })

  it('is the whole of it where everything requested came back', () => {
    expect(
      mountDetail({ did: 9, tested: 9, recalled: 9 }).get('[data-day-detail="recall-rate"]').text(),
    ).toBe('100% came back')
  })
})

describe('a day nobody answered on', () => {
  it('says so, and says nothing else', () => {
    const summary = mountDetail()
    expect(summary.get('[data-day-detail="count"]').text()).toBe('Nothing answered')
    expect(summary.find('[data-day-detail="ratings"]').exists()).toBe(false)
    expect(summary.find('[data-day-detail="recall-rate"]').exists()).toBe(false)
  })
})

describe('a day still to come', () => {
  it('says what falls on it and not what was done on it', () => {
    expect(mountDetail({ isAhead: true, did: 7 }).get('[data-day-detail="count"]').text()).toBe(
      '7 to come',
    )
  })

  // A day ahead was not answered on, so the four and the share are about
  // nothing.
  it('says none of the four and no share, whatever it carries', () => {
    const summary = mountDetail({ isAhead: true, did: 7, again: 2, tested: 4, recalled: 3 })
    expect(summary.find('[data-day-detail="ratings"]').exists()).toBe(false)
    expect(summary.find('[data-day-detail="recall-rate"]').exists()).toBe(false)
  })

  it('says nothing falls on a day nothing falls on', () => {
    expect(mountDetail({ isAhead: true }).get('[data-day-detail="count"]').text()).toBe(
      'Nothing answered',
    )
  })
})
