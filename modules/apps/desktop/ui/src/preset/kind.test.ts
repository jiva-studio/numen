/**
 * What one preset tab holds: the knob, the fields under it, and what is
 * written.
 *
 * The goal steers one value, and the knob and the field showing it are two
 * views of it. Every other field is a person's to take out of the goal's hands.
 */
import { describe, expect, it } from 'vitest'

import { presetting, type Said } from './kind'
import { fieldsUnder, nearest, standing, steers, type Field } from './curve'
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
  closed: '',
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
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
  overdue: 0,
  unbegun: 0,
  honest: true,
}

/** The day every test here runs on, so a goal of a date counts from one place. */
const NOW = new Date('2026-08-30T00:00:00Z')

/** The same range read as days, which is the goal a date steers. */
const dated: Curve = {
  ...curve,
  goal: 'date',
  grid: [10, 20, 30, 40],
  days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
  at: curve.at.map((one) => ({ ...one, minutes: 45 })),
}

/** A vault answering with one preset and that curve, and what was written into it. */
/** The file every test here opens, which stands where the curve's marks stand. */
const STEADY: Settings = { ...DEFAULTS, minutesADay: 20, reviewsADay: 80, retention: 0.88 }

const opened = async (
  settings: Partial<Settings> = {},
  answers: Curve | ((asked: Settings) => Curve) = curve,
) => {
  const written: Settings[] = []
  const asked: Goal[] = []
  const core: Presets = {
    read: async (path) => ({
      preset: { path, title: 'Steady', settings: { ...STEADY, ...settings }, problems: [] },
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
      return typeof answers === 'function' ? answers(put) : answers
    },
  }
  const host = { closes: () => {} } as unknown as Host
  const puts = { holds: () => {} } as unknown as Putting
  const kind = presetting(core, host, puts, () => {}, () => NOW)
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

  // The knob rides 25 places of a grid and a person types whatever they like.
  // What they typed is what the file gets; the knob only shows where that
  // leaves them.
  it('is the number typed, and the knob goes to the place nearest it', async () => {
    const { held, written } = await opened()
    held.types('minutesADay', 21)
    held.settles()
    await Promise.resolve()
    expect(held.place()).toBe(2)
    expect(held.settings().minutesADay).toBe(21)
    expect(written.at(-1)?.minutesADay).toBe(21)
  })

  it('is the target typed, whatever place of the grid stands nearest', async () => {
    const { held, written } = await opened(
      { goal: 'retention' },
      {
        ...curve,
        goal: 'retention',
        grid: [0.8, 0.85, 0.9, 0.95],
        now: { at: 1, value: 0.85, day: '' },
      },
    )
    held.types('retention', 0.873)
    held.settles()
    await Promise.resolve()
    expect(held.place()).toBe(1)
    expect(held.settings().retention).toBe(0.873)
    expect(written.at(-1)?.retention).toBe(0.873)
  })

  it('is the day under a goal of a date, and a day typed moves the knob', async () => {
    const { held, written } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '2026-09-29')
    held.settles()
    await Promise.resolve()
    expect(held.place()).toBe(2)
    expect(held.settings().byDate).toBe('2026-09-29')
    expect(written.at(-1)?.byDate).toBe('2026-09-29')
  })

  it('is the day typed, and not the day the nearest place of the grid names', async () => {
    const { held, written } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '2026-09-22')
    held.settles()
    await Promise.resolve()
    expect(held.place()).toBe(1)
    expect(held.settings().byDate).toBe('2026-09-22')
    expect(written.at(-1)?.byDate).toBe('2026-09-22')
  })

  it('leaves a field holding no day where it stands, and the knob where it is', async () => {
    const { held } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '')
    await Promise.resolve()
    expect(held.place()).toBe(0)
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

  // A goal already worked out is drawn again as it was, so moving between the
  // three is instant and never puts the picture back into its waiting state.
  it('is asked for once for each goal, however often they are moved between', async () => {
    const { held, asked } = await opened()
    const settle = async () => {
      for (let i = 0; i < 4; i += 1) await Promise.resolve()
    }

    held.chooses('retention')
    await settle()
    held.chooses('date')
    await settle()
    expect(asked).toStrictEqual(['minutes', 'retention', 'date'])

    held.chooses('minutes')
    await settle()
    expect(held.curve().honest).toBe(true)
    held.chooses('retention')
    await settle()
    expect(held.curve().honest).toBe(true)
    expect(asked).toStrictEqual(['minutes', 'retention', 'date'])
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

// The goal names one budget and moves that. Every other setting is the
// person's, and the control neither predicts it nor writes it.
describe('what the control writes', () => {
  it('is the minutes alone under a goal of minutes, whatever the counts hold', async () => {
    // The case reported: a day of minutes the card limits had been cut to
    // nothing under.
    const { held, written } = await opened({ minutesADay: 34, newADay: 12, reviewsADay: 0 })
    held.moves(3)
    held.settles()
    await Promise.resolve()

    expect(held.settings().minutesADay).toBe(30)
    expect(written.at(-1)?.minutesADay).toBe(30)
    expect(written.at(-1)?.newADay).toBe(12)
    expect(written.at(-1)?.reviewsADay).toBe(0)
    expect(written.at(-1)?.retention).toBe(0.88)
  })

  it('leaves the counts and the target alone under a goal of minutes', async () => {
    const { held, written } = await opened({ reviewsADay: 12, retention: 0.95 })
    held.moves(1)
    held.settles()
    await Promise.resolve()
    expect(held.settings().reviewsADay).toBe(12)
    expect(held.settings().retention).toBe(0.95)
    expect(written.at(-1)?.reviewsADay).toBe(12)
    expect(written.at(-1)?.retention).toBe(0.95)
  })
})

// A setting the goal does not name is never deleted and never zeroed: it waits
// in the file for the goal that names it to come round again.
describe('a setting the goal on screen does not name', () => {
  it('is not drawn, and keeps its value in the file across a write', async () => {
    const { held, written } = await opened({ minutesADay: 34, newADay: 12, reviewsADay: 7 })
    expect(fieldsUnder(held.settings().goal, held.settings().learned)).not.toContain('reviewsADay')

    held.moves(1)
    held.settles()
    await Promise.resolve()
    expect(written.at(-1)?.reviewsADay).toBe(7)
    expect(written.at(-1)?.newADay).toBe(12)
  })

  it('comes back the moment its own goal is chosen again', async () => {
    const { held, written } = await opened({ minutesADay: 34, newADay: 12, reviewsADay: 7 })
    held.moves(3)
    held.settles()
    await Promise.resolve()

    held.chooses('retention')
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(fieldsUnder('retention', DEFAULTS.learned)).toContain('reviewsADay')
    expect(held.settings().reviewsADay).toBe(7)
    expect(held.settings().newADay).toBe(12)
    expect(written.at(-1)?.reviewsADay).toBe(7)
  })
})

/**
 * A curve worked out off the settings it was asked under, the way the
 * application works one out: its range runs to what carrying the load costs,
 * which the share of the day going to the debt moves.
 */
const ranging = (asked: Settings): Curve => {
  const top = 60 + asked.backlog
  const grid = Array.from({ length: 7 }, (_, at) => Math.round((top * (at + 1)) / 7))
  const value = standing(asked, NOW)
  return {
    ...curve,
    goal: asked.goal,
    grid,
    at: grid.map((minutes) => point({ minutes, reviews: minutes * 4 })),
    now: { at: nearest(grid, value), value, day: '' },
    suggested: { at: grid.length - 1, value: grid[grid.length - 1] ?? 0, day: '' },
  }
}

// Nothing but the field a person types in may change, and the budget the goal
// steers least of all: it is what the picture is scaled to.
describe('a field the goal does not steer, typed', () => {
  // A curve asked for under a new share of the day comes back over a range of
  // its own, and a place of the old grid stands at another value on it. The
  // knob goes back to where the preset stands, so the picture and the file say
  // one thing and the next touch of the knob writes what the file already says.
  it('leaves the knob standing where the preset stands, whatever range comes back', async () => {
    const { held, written } = await opened({ minutesADay: 23 }, ranging)
    expect(held.place()).toBe(held.curve().now.at)

    held.types('backlog', 5)
    held.settles()
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(held.place()).toBe(held.curve().now.at)
    expect(held.settings().minutesADay).toBe(23)
    expect(written.at(-1)?.minutesADay).toBe(23)
  })

  const SAID: Partial<Record<Field, Said>> = {
    newADay: 4,
    reviewsADay: 33,
    counts: 'shows',
    backlog: 5,
    load: { sat: 50 },
    evenLoad: false,
  }

  it('leaves every other setting exactly as it stood, under every goal', async () => {
    for (const goal of ['minutes', 'retention'] as const) {
      for (const field of fieldsUnder(goal, DEFAULTS.learned)) {
        const said = SAID[field]
        if (field === steers(goal) || said === undefined) continue

        const { held, written } = await opened({ goal, minutesADay: 23 }, ranging)
        const was = held.settings()
        held.types(field, said)
        held.settles()
        await Promise.resolve()
        await Promise.resolve()
        await Promise.resolve()

        const now = held.settings()
        expect({ ...now, [field]: was[field] }).toStrictEqual(was)
        expect(written.at(-1)).toStrictEqual(now)
      }
    }
  })
})

