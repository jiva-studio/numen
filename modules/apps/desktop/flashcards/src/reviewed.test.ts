import { describe, expect, it } from 'vitest'

import { reviewed } from './reviewed'
import type { Asks, Said } from './reviewed'

const said = (days: [string, number][], streak = 0, due: [string, number][] = []): Said => ({
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
    const cards: Asks = {
      async listReviewDays() {
        return said(
          [
            ['2026-08-28', 12],
            ['2026-08-29', 3],
          ],
          2,
        )
      },
    }
    const one = reviewed({ cards, failed: () => {} })

    await one.read('01VAULT')

    expect(one.days.value.get('2026-08-28')?.answered).toBe(12)
    expect(one.streak.value).toBe(2)
    expect(one.answered.value).toBe(15)
    expect(one.of.value).toBe('01VAULT')
  })

  // A person who moved on while the answer was on its way is looking at another
  // vault, and these days are not its days.
  it('is dropped when the person has moved to another vault', async () => {
    let settle = (_: Said) => {}
    const cards: Asks = {
      listReviewDays({ vault }) {
        if (vault === '01SLOW') {
          return new Promise<Said>((then) => {
            settle = then
          })
        }
        return Promise.resolve(said([['2026-08-29', 1]], 1))
      },
    }
    const one = reviewed({ cards, failed: () => {} })

    const slow = one.read('01SLOW')
    await one.read('01OTHER')
    settle(said([['2020-01-01', 99]], 40))
    await slow

    expect(one.days.value.has('2020-01-01')).toBe(false)
    expect(one.streak.value).toBe(1)
    expect(one.of.value).toBe('01OTHER')
  })

  it('holds what is still to come apart from what was done', async () => {
    const cards: Asks = {
      async listReviewDays() {
        return said([['2026-08-29', 3]], 1, [
          ['2026-08-31', 12],
          ['2026-09-05', 4],
        ])
      },
    }
    const one = reviewed({ cards, failed: () => {} })

    await one.read('01VAULT')

    expect(one.due.value.get('2026-08-31')).toBe(12)
    expect(one.days.value.has('2026-08-31')).toBe(false)
    expect(one.due.value.has('2026-08-29')).toBe(false)
  })

  it('is nothing for no vault at all', async () => {
    const cards: Asks = { listReviewDays: () => Promise.reject(new Error('never asked')) }
    const one = reviewed({ cards, failed: () => {} })

    await one.read('')

    expect(one.days.value.size).toBe(0)
    expect(one.of.value).toBe('')
  })

  it('says what went wrong and holds nothing', async () => {
    const trouble: unknown[] = []
    const cards: Asks = { listReviewDays: () => Promise.reject(new Error('no such vault')) }
    const one = reviewed({ cards, failed: (why) => trouble.push(why) })

    await one.read('01VAULT')

    expect(trouble).toHaveLength(1)
    expect(one.days.value.size).toBe(0)
    expect(one.streak.value).toBe(0)
  })
})
