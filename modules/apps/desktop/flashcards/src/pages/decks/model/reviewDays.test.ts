import { describe, expect, it } from 'vitest'

import { useReviewDays } from './reviewDays'
import type { ReviewDays, ReviewDaysClient } from './reviewDays'

const createReviewDays = (
  days: [string, number][],
  streak = 0,
  due: [string, number][] = [],
): ReviewDays => ({
  days: days.map(([day, answered]) => ({
    day,
    answered,
    again: 0,
    hard: 0,
    good: answered,
    easy: 0,
    asked: answered,
    recalled: answered,
  })),
  due: due.map(([day, answered]) => ({ day, answered })),
  streak,
  answered: days.reduce((sum, [, answered]) => sum + answered, 0),
})

describe('what a vault was answered on', () => {
  it('is held by the day it was answered on', async () => {
    const cards: ReviewDaysClient = {
      async listReviewDays() {
        return createReviewDays(
          [
            ['2026-08-28', 12],
            ['2026-08-29', 3],
          ],
          2,
        )
      },
    }
    const one = useReviewDays({ cards, reportError: () => {}, widest: () => 640 })

    await one.read('01VAULT')

    expect(one.days.value.get('2026-08-28')?.answered).toBe(12)
    expect(one.streak.value).toBe(2)
    expect(one.answered.value).toBe(15)
    expect(one.of.value).toBe('01VAULT')
  })

  // A person who moved on while the answer was on its way is looking at another
  // vault, and these days are not its days.
  it('is dropped when the person has moved to another vault', async () => {
    let settle = (_: ReviewDays) => {}
    const cards: ReviewDaysClient = {
      listReviewDays({ vault }) {
        if (vault === '01SLOW') {
          return new Promise<ReviewDays>((then) => {
            settle = then
          })
        }
        return Promise.resolve(createReviewDays([['2026-08-29', 1]], 1))
      },
    }
    const one = useReviewDays({ cards, reportError: () => {}, widest: () => 640 })

    const slow = one.read('01SLOW')
    await one.read('01OTHER')
    settle(createReviewDays([['2020-01-01', 99]], 40))
    await slow

    expect(one.days.value.has('2020-01-01')).toBe(false)
    expect(one.streak.value).toBe(1)
    expect(one.of.value).toBe('01OTHER')
  })

  it('holds what is still to come apart from what was done', async () => {
    const cards: ReviewDaysClient = {
      async listReviewDays() {
        return createReviewDays([['2026-08-29', 3]], 1, [
          ['2026-08-31', 12],
          ['2026-09-05', 4],
        ])
      },
    }
    const one = useReviewDays({ cards, reportError: () => {}, widest: () => 640 })

    await one.read('01VAULT')

    expect(one.due.value.get('2026-08-31')).toBe(12)
    expect(one.days.value.has('2026-08-31')).toBe(false)
    expect(one.due.value.has('2026-08-29')).toBe(false)
  })

  it('is nothing for no vault at all', async () => {
    const cards: ReviewDaysClient = {
      listReviewDays: () => Promise.reject(new Error('never asked')),
    }
    const one = useReviewDays({ cards, reportError: () => {}, widest: () => 640 })

    await one.read('')

    expect(one.days.value.size).toBe(0)
    expect(one.of.value).toBe('')
  })

  it('says what went wrong and holds nothing', async () => {
    const errors: unknown[] = []
    const cards: ReviewDaysClient = {
      listReviewDays: () => Promise.reject(new Error('no such vault')),
    }
    const one = useReviewDays({ cards, reportError: (why) => errors.push(why), widest: () => 640 })

    await one.read('01VAULT')

    expect(errors).toHaveLength(1)
    expect(one.days.value.size).toBe(0)
    expect(one.streak.value).toBe(0)
  })
})

// A vault grows a day for every day it is reviewed, and the grid draws the
// stretch it has room for. The ask is that stretch and not the whole of it.
describe('the days asked about', () => {
  it('is the stretch the widest grid could draw, and not every day there is', async () => {
    let asked: { vault: string; from: string; to: string } | null = null
    const cards: ReviewDaysClient = {
      listReviewDays(request) {
        asked = request
        return Promise.resolve({ days: [], due: [], streak: 0, answered: 0 })
      },
    }
    const one = useReviewDays({ cards, reportError: () => {}, widest: () => 640 })

    await one.read('01VAULT')

    expect(asked).not.toBeNull()
    const said = asked as unknown as { from: string; to: string }
    expect(said.from).not.toBe('')
    expect(said.to).not.toBe('')
    expect(said.from < said.to).toBe(true)

    // A grid with less room asks about less time.
    const narrow = useReviewDays({ cards, reportError: () => {}, widest: () => 200 })
    await narrow.read('01VAULT')
    const less = asked as unknown as { from: string; to: string }
    expect(less.from > said.from).toBe(true)
  })
})
