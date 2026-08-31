/**
 * What one preset tab holds: the knob, the fields under it, and what is
 * written.
 *
 * The goal steers one value, and the knob and the field showing it are two
 * views of it. Every other field is a person's to take out of the goal's hands.
 */
import { describe, expect, it } from 'vitest'

import { presetting } from './kind'
import { DEFAULTS, type Curve, type Goal, type Point, type Presets, type Settings } from './core'
import type { Host } from '../windowing'
import type { Putting } from '../putting'

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  met: true,
  ...over,
})

const curve: Curve = {
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [
    point(),
    point({ reviews: 40, retained: 0.8 }),
    point({ reviews: 80, retained: 0.88 }),
    point({ reviews: 120, retained: 0.93 }),
  ],
  now: { at: 2, value: 20, day: '' },
  suggested: { at: 3, value: 30, day: '' },
  decks: 1,
  cards: 400,
  honest: true,
}

/** The same range read as days, which is the goal a date steers. */
const dated: Curve = {
  ...curve,
  goal: 'date',
  grid: [10, 20, 30, 40],
  days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
  at: curve.at.map((one) => ({ ...one, minutes: 45 })),
}

/** A vault answering with one preset and that curve, and what was written into it. */
/**
 * A file whose values are the ones its goal produces where it stands. It is
 * the file of a preset following its goal, so nothing in it is marked.
 */
const AGREEING: Settings = { ...DEFAULTS, minutesADay: 20, reviewsADay: 80, retention: 0.88 }

const opened = async (settings: Partial<Settings> = {}, answers: Curve = curve) => {
  const written: Settings[] = []
  const asked: Goal[] = []
  const core: Presets = {
    read: async (path) => ({
      preset: { path, title: 'Steady', settings: { ...AGREEING, ...settings }, problems: [] },
      refusal: null,
      at: 'one',
    }),
    scheduling: async () => ({ preset: null, refusal: null, at: '' }),
    write: async (_path, put) => {
      written.push(put)
      return { refusal: null, changed: false, at: 'two' }
    },
    curve: async (_path, put) => {
      asked.push(put.goal)
      return answers
    },
  }
  const host = { closes: () => {} } as unknown as Host
  const puts = { holds: () => {} } as unknown as Putting
  const kind = presetting(core, host, puts, () => {})
  kind.kind.opens('Steady.md')
  const held = kind.holds('Steady.md')
  // The read and the curve behind it are two answers, and both are awaited.
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
  return { held, written, asked, changed: kind.changed }
}

describe('the value the goal steers', () => {
  it('leaves the knob, the field and what is written at one value after a drag', async () => {
    const { held, written } = await opened()
    held.moves(3)
    held.settles()
    await Promise.resolve()
    expect(held.place()).toBe(3)
    expect(held.settings().minutesADay).toBe(30)
    expect(written.at(-1)?.minutesADay).toBe(30)
  })

  it('is the knob moved when it is typed into, and does not leave the goal', async () => {
    const { held, written } = await opened()
    held.types('minutesADay', 21)
    await Promise.resolve()
    expect(held.place()).toBe(2)
    expect(held.settings().minutesADay).toBe(20)
    expect(held.byHand().has('minutesADay')).toBe(false)
    expect(written.at(-1)?.minutesADay).toBe(20)
  })

  it('is the day under a goal of a date, and a day typed moves the knob', async () => {
    const { held, written } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '2026-09-29')
    await Promise.resolve()
    expect(held.place()).toBe(2)
    expect(held.settings().byDate).toBe('2026-09-29')
    expect(held.byHand().has('byDate')).toBe(false)
    expect(written.at(-1)?.byDate).toBe('2026-09-29')
  })

  it('leaves a field holding no day where it stands, and the knob where it is', async () => {
    const { held } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '')
    await Promise.resolve()
    expect(held.place()).toBe(0)
    expect(held.byHand().has('byDate')).toBe(false)
  })
})

describe('the curve behind the knob', () => {
  it('is asked for once for the goal, and a walk of the grid asks nothing', async () => {
    const { held, asked } = await opened()
    held.moves(1)
    held.moves(3)
    held.settles()
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()
    expect(asked).toStrictEqual(['minutes'])
  })

  it('is asked for again where a field the knob does not ride is typed', async () => {
    const { held, asked } = await opened()
    held.types('newADay', 4)
    await Promise.resolve()
    expect(asked).toStrictEqual(['minutes', 'minutes'])
  })

  it('is left as it stands where the file comes back saying what it already says', async () => {
    const { held, asked, changed } = await opened()
    held.moves(3)
    changed(['Steady.md'])
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()
    expect(asked).toStrictEqual(['minutes'])
    expect(held.place()).toBe(3)
  })
})

describe('a field the goal fills in itself', () => {
  it('keeps what a person typed, and the knob no longer writes it', async () => {
    const { held, written } = await opened()
    held.types('reviewsADay', 12)
    await Promise.resolve()
    expect(held.byHand().has('reviewsADay')).toBe(true)
    held.moves(3)
    held.settles()
    await Promise.resolve()
    expect(held.settings().reviewsADay).toBe(12)
    expect(held.settings().retention).toBe(0.93)
    expect(written.at(-1)?.reviewsADay).toBe(12)
  })

  it('follows the goal again when it is handed back', async () => {
    const { held } = await opened()
    held.types('reviewsADay', 12)
    await Promise.resolve()
    held.follows('reviewsADay')
    expect(held.byHand().has('reviewsADay')).toBe(false)
    expect(held.settings().reviewsADay).toBe(80)
  })
})

// Nothing is remembered: the file carries the goal and the values, and a field
// is off the goal exactly while the two disagree.
describe('which fields are off the goal', () => {
  it('is none of them, for a file whose values are the ones its goal produces', async () => {
    const { held } = await opened()
    expect([...held.byHand()]).toStrictEqual([])
  })

  it('is the rows a file disagrees with its goal on, and no others', async () => {
    const { held } = await opened({ reviewsADay: 12 })
    expect([...held.byHand()]).toStrictEqual(['reviewsADay'])
  })

  it('is a field the goal does not fill in itself under no circumstance', async () => {
    const { held } = await opened({ newADay: 4, evenLoad: false })
    expect([...held.byHand()]).toStrictEqual([])
  })

  it('is what it was after a write, and after the file is read again', async () => {
    const { held, changed } = await opened({ reviewsADay: 12 })
    held.settles()
    await Promise.resolve()
    expect([...held.byHand()]).toStrictEqual(['reviewsADay'])
    changed(['Steady.md'])
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()
    expect([...held.byHand()]).toStrictEqual(['reviewsADay'])
  })

  it('loses the row handed back, and the goal’s own value is written for it', async () => {
    const { held, written } = await opened({ reviewsADay: 12 })
    held.follows('reviewsADay')
    await Promise.resolve()
    expect([...held.byHand()]).toStrictEqual([])
    expect(written.at(-1)?.reviewsADay).toBe(80)
  })
})
