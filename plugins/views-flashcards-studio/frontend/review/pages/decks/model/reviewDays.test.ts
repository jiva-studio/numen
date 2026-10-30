import { describe, expect, it } from 'vitest'

import { useReviewDays } from './reviewDays'
import type { ReviewDays, ReviewDaysClient } from './reviewDays'

const createReviewDays = (
  days: [string, number][],
  streak = 0,
  due: [string, number][] = [],
): ReviewDays => ({
  days: days.map(([day, reviewed]) => ({
    day,
    reviewed,
    again: 0,
    hard: 0,
    good: reviewed,
    easy: 0,
    tested: reviewed,
    recalled: reviewed,
  })),
  due: due.map(([day, reviewed]) => ({ day, reviewed })),
  streak,
  reviewed: days.reduce((sum, [, reviewed]) => sum + reviewed, 0),
})

describe('what a vault was answered on', () => {
  it('is retained by the day it was answered on', async () => {
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
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })

    await one.read('01VAULT')

    expect(one.days.value.get('2026-08-28')?.reviewed).toBe(12)
    expect(one.streak.value).toBe(2)
    expect(one.answerCount.value).toBe(15)
    expect(one.of.value).toBe('01VAULT')
  })

  // A user who switched to another vault while the response was pending is looking
  // at another vault, and these days do not belong to it.
  it('is dropped when the person has moved to another vault', async () => {
    let settle: (_: ReviewDays) => void = (_: ReviewDays) => undefined
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
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })

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
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })

    await one.read('01VAULT')

    expect(one.due.value.get('2026-08-31')).toBe(12)
    expect(one.days.value.has('2026-08-31')).toBe(false)
    expect(one.due.value.has('2026-08-29')).toBe(false)
  })

  it('is nothing for no vault at all', async () => {
    const cards: ReviewDaysClient = {
      listReviewDays: () => Promise.reject(new Error('never asked')),
    }
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })

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

// A vault gains a day for every day it is reviewed, and the grid renders the
// range it has space for. The request covers that range and not the whole history.
describe('the days requested about', () => {
  it('is the stretch the widest grid could draw, and not every day there is', async () => {
    let request: { vault: string; from: string; to: string } | null = null
    const cards: ReviewDaysClient = {
      listReviewDays(query) {
        request = query
        return Promise.resolve({ days: [], due: [], streak: 0, reviewed: 0 })
      },
    }
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })

    await one.read('01VAULT')

    expect(request).not.toBeNull()
    const response = request as unknown as { from: string; to: string }
    expect(response.from).not.toBe('')
    expect(response.to).not.toBe('')
    expect(response.from < response.to).toBe(true)

    // A grid with less space requests a shorter range.
    const narrow = useReviewDays({ cards, reportError: () => undefined, widest: () => 200 })
    await narrow.read('01VAULT')
    const less = request as unknown as { from: string; to: string }
    expect(less.from > response.from).toBe(true)
  })
})

describe('what is retained of a vault after another answer', () => {
  it('holds nothing, and asks nothing, once it is read for no vault at all', async () => {
    const request: string[] = []
    const errors: unknown[] = []
    const cards: ReviewDaysClient = {
      async listReviewDays({ vault }) {
        request.push(vault)
        return createReviewDays([['2026-08-29', 3]], 1, [['2026-08-31', 12]])
      },
    }
    const one = useReviewDays({ cards, reportError: (why) => errors.push(why), widest: () => 640 })
    await one.read('01VAULT')

    await one.read('')

    expect(request).toEqual(['01VAULT'])
    expect(errors).toHaveLength(0)
    expect(one.days.value.size).toBe(0)
    expect(one.due.value.size).toBe(0)
    expect(one.streak.value).toBe(0)
    expect(one.answerCount.value).toBe(0)
    expect(one.of.value).toBe('')
  })

  it('drops what it retained of a vault whose days could not be read again', async () => {
    let fails = false
    const cards: ReviewDaysClient = {
      async listReviewDays() {
        if (fails) throw new Error('no such vault')
        return createReviewDays([['2026-08-29', 3]], 1, [['2026-08-31', 12]])
      },
    }
    const one = useReviewDays({ cards, reportError: () => undefined, widest: () => 640 })
    await one.read('01VAULT')

    fails = true
    await one.read('01VAULT')

    expect(one.days.value.size).toBe(0)
    expect(one.due.value.size).toBe(0)
    expect(one.streak.value).toBe(0)
    expect(one.answerCount.value).toBe(0)
  })

  it('keeps the vault moved to, and says nothing, when the one left fails', async () => {
    let refuse: (_: Error) => void = (_: Error) => undefined
    const errors: unknown[] = []
    const cards: ReviewDaysClient = {
      listReviewDays({ vault }) {
        if (vault === '01SLOW') {
          return new Promise<ReviewDays>((_, then) => {
            refuse = then
          })
        }
        return Promise.resolve(createReviewDays([['2026-08-29', 1]], 1))
      },
    }
    const one = useReviewDays({ cards, reportError: (why) => errors.push(why), widest: () => 640 })

    const slow = one.read('01SLOW')
    await one.read('01OTHER')
    refuse(new Error('no such vault'))
    await slow

    expect(errors).toHaveLength(0)
    expect(one.days.value.get('2026-08-29')?.reviewed).toBe(1)
    expect(one.of.value).toBe('01OTHER')
  })
})
