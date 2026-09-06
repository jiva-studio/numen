/**
 * What one preset tab holds: the knob, the fields under it, and what is
 * written.
 *
 * The goal steers one value, and the knob and the field showing it are two
 * views of it. Every other field is a person's to take out of the goal's hands.
 */
import { describe, expect, it } from 'vitest'
import { StopReason } from '@numen/protocol'

import { presetting, type SettingValue } from './kind'
import { fieldsUnder, goalValue, nearest, steers, type Field } from './curve'
import {
  DEFAULTS,
  NO_BOUNDS,
  type Curve,
  type Goal,
  type Point,
  type Presets,
  type ReadResult,
  type Settings,
  type WriteResult,
} from './core'
import { BOUNDS } from './drawn'
import type { RefusalReason } from '../core'
import type { Host } from '../tabs/windowing'
import type { FileOpeners } from '../tabs/openers'
import { WORDS as words } from './words'

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  closed: [],
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
  answers: Curve | ((asked: Settings) => Curve | Promise<Curve>) = curve,
  reading: (time: number) => Partial<ReadResult> = () => ({}),
  writing: (time: number) => Partial<WriteResult> | Promise<Partial<WriteResult>> = () => ({}),
) => {
  const written: Settings[] = []
  const asked: Goal[] = []
  const closed: string[] = []
  let times = 0
  let writes = 0
  const core: Presets = {
    read: async (path) => ({
      preset: {
        path,
        title: 'Steady',
        settings: { ...STEADY, ...settings },
        problems: [],
        stops: StopReason.NOTHING,
        stopsOn: StopReason.NOTHING,
      },
      refusal: null,
      at: 'one',
      bounds: BOUNDS,
      ...reading(times++),
    }),
    scheduling: async () => ({ preset: null, refusal: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    makes: async () => ({ path: '', refusal: null }),
    schedules: async () => ({ refusal: null, changed: false, at: '' }),
    write: async (_path, put) => {
      written.push(put)
      return { refusal: null, changed: false, at: 'two', ...(await writing(writes++)) }
    },
    curve: async (_path, put) => {
      asked.push(put.goal)
      return typeof answers === 'function' ? answers(put) : answers
    },
  }
  const host = { closes: (tab: string) => void closed.push(tab) } as unknown as Host
  const puts = { holds: () => {} } as unknown as FileOpeners
  const kind = presetting(core, host, puts, () => {}, () => NOW)
  const held = await kind.kind.opens('Steady.md')
  // The read and the curve behind it are two answers, and both are awaited.
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
  return {
    held,
    written,
    asked,
    closed,
    holds: kind.holds,
    changed: kind.changed,
    flush: kind.flush,
  }
}

/** A moment for the read, the curve behind it and a write to land. */
const after = async () => {
  for (let i = 0; i < 10; i += 1) await Promise.resolve()
}

/**
 * A tab whose first read is still out. Until the test lets it land, the tab is
 * standing at the placeholder settings and not at the file's own.
 */
const opening = async (file: Partial<Settings>) => {
  let lands = () => {}
  const held = new Promise<void>((settle) => {
    lands = settle
  })
  const written: Settings[] = []
  const core: Presets = {
    read: async (path) => {
      await held
      return {
        preset: {
          path,
          title: 'Steady',
          settings: { ...STEADY, ...file },
          problems: [],
          stops: StopReason.NOTHING,
          stopsOn: StopReason.NOTHING,
        },
        refusal: null,
        at: 'one',
        bounds: BOUNDS,
      }
    },
    scheduling: async () => ({ preset: null, refusal: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    makes: async () => ({ path: '', refusal: null }),
    schedules: async () => ({ refusal: null, changed: false, at: '' }),
    write: async (_path, put) => {
      written.push(put)
      return { refusal: null, changed: false, at: 'two' }
    },
    curve: async () => curve,
  }
  const host = { closes: () => {} } as unknown as Host
  const puts = { holds: () => {} } as unknown as FileOpeners
  const kind = presetting(core, host, puts, () => {}, () => NOW)
  return { tab: await kind.kind.opens('Steady.md'), written, lands }
}

describe('the value the goal steers', () => {
  it('leaves the knob, the field and what is written at one value after a drag', async () => {
    const { held, written } = await opened()
    held.moves(3)
    held.settles()
    await Promise.resolve()
    expect(held.place.value).toBe(3)
    expect(held.settings.value.minutesADay).toBe(30)
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
    expect(held.place.value).toBe(2)
    expect(held.settings.value.minutesADay).toBe(21)
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
    expect(held.place.value).toBe(1)
    expect(held.settings.value.retention).toBe(0.873)
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
    expect(held.place.value).toBe(2)
    expect(held.settings.value.byDate).toBe('2026-09-29')
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
    expect(held.place.value).toBe(1)
    expect(held.settings.value.byDate).toBe('2026-09-22')
    expect(written.at(-1)?.byDate).toBe('2026-09-22')
  })

  it('leaves a field holding no day where it stands, and the knob where it is', async () => {
    const { held } = await opened(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { at: 0, value: 10, day: '2026-09-09' } },
    )
    held.types('byDate', '')
    await Promise.resolve()
    expect(held.place.value).toBe(0)
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

  // What the material comes to is a fact about the vault: no setting moves one
  // of those figures, so they are kept beside the curve and stand at what the
  // last answer counted them while the next one is out.
  it('keeps what the material comes to while a curve is on its way', async () => {
    const { held } = await opened()
    expect(held.material.value).toStrictEqual({ decks: 1, cards: 400, overdue: 0, unbegun: 0 })

    held.types('newADay', 4)
    expect(held.waiting.value).toBe(true)
    expect(held.curve.value.honest).toBe(false)
    expect(held.material.value).toStrictEqual({ decks: 1, cards: 400, overdue: 0, unbegun: 0 })
  })

  it('counts the material at nothing until an answer has counted it', async () => {
    const { held } = await opened({}, () => new Promise<Curve>(() => {}))
    expect(held.material.value).toBeNull()
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
    expect(held.curve.value.honest).toBe(true)
    held.chooses('retention')
    await settle()
    expect(held.curve.value.honest).toBe(true)
    expect(asked).toStrictEqual(['minutes', 'retention', 'date'])
  })

  it('leaves the knob where it stands where the file says what it already said', async () => {
    const { held, changed } = await opened()
    held.moves(3)
    changed(['Steady.md'])
    await after()
    expect(held.place.value).toBe(3)
  })

  // The counts behind a curve are the vault's, not the preset's, so a file read
  // again is a curve to ask for again however little the settings moved.
  it('is asked afresh on a re-read, so a deck pointed here since is seen', async () => {
    let decks = 0
    const { held, changed } = await opened({}, () => ({ ...curve, decks: decks++ }))
    expect(held.curve.value.decks).toBe(0)

    changed(['Steady.md'])
    await after()
    expect(held.curve.value.decks).toBe(1)
  })

  // The range the line is drawn over runs to the value the knob rides, so a
  // value typed past the end of it is a curve nobody has been answered.
  it('is asked for again where a value past the end of the range is typed', async () => {
    const reaching = (asked: Settings): Curve => ({
      ...curve,
      grid: [0, asked.minutesADay / 2, asked.minutesADay],
      at: [point(), point(), point()],
      now: { at: 2, value: asked.minutesADay, day: '' },
    })
    const { held, asked } = await opened({}, reaching)
    expect(held.curve.value.grid.at(-1)).toBe(20)

    held.types('minutesADay', 120)
    await after()
    expect(asked).toStrictEqual(['minutes', 'minutes'])
    expect(held.curve.value.grid.at(-1)).toBe(120)
  })

  // The figures over the picture are read as the answer to what stands on
  // screen, so the run of a settled question is nobody's answer to a new one.
  it('is nobody’s answer while the answer to the settings now standing is out', async () => {
    const { held } = await opened()
    expect(held.curve.value.honest).toBe(true)

    held.types('newADay', 4)
    expect(held.curve.value.honest).toBe(false)
    await after()
    expect(held.curve.value.honest).toBe(true)
  })
})

// The goal is the one choice a person makes outright, and it is theirs the
// moment they make it. A window shut before an answer lands loses nothing.
describe('the goal chosen', () => {
  it('is written without waiting on the curve behind it', async () => {
    const { held, written } = await opened({}, () => new Promise<Curve>(() => {}))
    held.chooses('retention')
    await after()
    expect(written.at(-1)?.goal).toBe('retention')
  })

  it('names a day where the file names none, since a date is aimed at one', async () => {
    const { held, written } = await opened({ byDate: '' }, dated)
    held.chooses('date')
    await after()
    expect(held.settings.value.byDate).toBe('2026-09-29')
    expect(written.at(-1)?.byDate).toBe('2026-09-29')
  })

  it('keeps the day the file names, whether it is ahead of today or behind', async () => {
    for (const day of ['2026-12-25', '2026-08-30', '2026-01-06']) {
      const { held, written } = await opened({ byDate: day }, dated)
      held.chooses('date')
      await after()
      expect(held.settings.value.byDate).toBe(day)
      expect(written.at(-1)?.byDate).toBe(day)
    }
  })
})

// A tab is the one thing that holds a preset. Anything else answering for one
// answers with settings nobody set, and writes them into the file at the first
// control let go of.
describe('a preset no tab has open', () => {
  it('is held by nothing, where no tab ever opened it', async () => {
    const { holds } = await opened()
    expect(holds('Nowhere.md')).toBeUndefined()
  })

  it('is what a preset becomes once its tab is shut', async () => {
    const { held, holds } = await opened()
    held.shuts('Steady.md')
    await after()
    expect(holds('Steady.md')).toBeUndefined()
  })

  it('is what a renamed preset becomes once its tab is shut', async () => {
    const { held, holds, changed } = await opened()
    changed([], [{ from: 'Steady.md', to: 'Slow.md' }])
    held.shuts('Slow.md')
    await after()
    expect(holds('Slow.md')).toBeUndefined()
    expect(holds('Steady.md')).toBeUndefined()
  })
})

// A person who is told what happened can do something about it. One sentence
// over every refusal names none of them.
describe('what the tab says it was refused for', () => {
  const refusals: readonly RefusalReason[] = ['missing', 'tooLarge', 'notANote', 'unreadable']

  it('is a sentence of its own for each refusal a read answers', async () => {
    const said: string[] = []
    for (const refusal of refusals) {
      const { held } = await opened({}, curve, () => ({ preset: null, refusal }))
      said.push(held.saying.value)
    }
    expect(said.every((one) => one !== '')).toBe(true)
    expect(new Set(said).size).toBe(refusals.length)
  })

  it('is a sentence of its own for each refusal a write answers', async () => {
    const said: string[] = []
    for (const refusal of refusals) {
      const { held } = await opened({}, curve, () => ({}), () => ({ refusal }))
      held.types('newADay', 4)
      held.settles()
      await after()
      said.push(held.saying.value)
    }
    expect(said.every((one) => one !== '')).toBe(true)
    expect(new Set(said).size).toBe(refusals.length)
  })

  // The settings are in the tab and nowhere else once a write is refused, and
  // a person deciding what to do next has to be told that.
  it('says the settings are still here where a write was refused', async () => {
    for (const refusal of refusals) {
      expect(words.notSaved(refusal)).toContain('still here')
    }
  })

  it('says the refusal of a read and the refusal of a write in different words', async () => {
    for (const refusal of refusals) {
      expect(words.refused(refusal)).not.toBe(words.notSaved(refusal))
    }
  })
})

// A picture that says it is reading the vault says an answer is on its way.
// Where none is coming, the tab says what happened and stops saying it.
describe('a curve nobody answers', () => {
  it('leaves the tab saying why, and not saying it is reading', async () => {
    const { held } = await opened({}, () => Promise.reject(new Error('the vault is gone')))
    expect(held.waiting.value).toBe(false)
    expect(held.saying.value).not.toBe('')
  })

  it('is what a file refused leaves, so no answer is waited on', async () => {
    const { held } = await opened({}, curve, () => ({ preset: null, refusal: 'notAPreset' }))
    expect(held.waiting.value).toBe(false)
    expect(held.saying.value).not.toBe('')
  })

  it('is waited on again where the goal is moved to one nobody has answered', async () => {
    const { held } = await opened({}, () => new Promise<Curve>(() => {}))
    held.chooses('retention')
    await after()
    expect(held.waiting.value).toBe(true)
  })
})

// A setting typed is written when the control is let go of, so at any moment
// the last of it stands in the tab and nowhere else. The tab answers for it
// where it is asked to go, and where the window is.
describe('what a tab still owes the file', () => {
  it('is written before the tab goes', async () => {
    const { held, written, closed } = await opened()
    held.types('newADay', 4)
    held.shuts('Steady.md')
    await after()
    expect(written.at(-1)?.newADay).toBe(4)
    expect(closed).toStrictEqual(['Steady.md'])
  })

  it('keeps the tab open where the write was refused, and says why', async () => {
    const { held, closed } = await opened({}, curve, () => ({}), () => ({ refusal: 'notAPreset' }))
    held.types('newADay', 4)
    held.shuts('Steady.md')
    await after()
    expect(closed).toStrictEqual([])
    expect(held.saying.value).not.toBe('')
  })

  it('lets the tab go the second time it is asked, the person having been told', async () => {
    const { held, closed } = await opened({}, curve, () => ({}), () => ({ refusal: 'notAPreset' }))
    held.types('newADay', 4)
    held.shuts('Steady.md')
    await after()
    held.shuts('Steady.md')
    await after()
    expect(closed).toStrictEqual(['Steady.md'])
  })

  it('keeps the tab open where the file moved under it and nothing was written', async () => {
    const { held, closed } = await opened({}, curve, () => ({}), () => ({ changed: true }))
    held.types('newADay', 4)
    held.shuts('Steady.md')
    await after()
    expect(closed).toStrictEqual([])
    expect(held.changed.value).toBe(true)
  })

  it('is written when the window goes', async () => {
    const { held, written, flush } = await opened()
    held.types('newADay', 4)
    await flush()
    expect(written.at(-1)?.newADay).toBe(4)
  })

  it('is waited for by the window going, where a write is already out', async () => {
    let lands = () => {}
    const { held, flush } = await opened({}, curve, () => ({}), (time) =>
      time === 0
        ? new Promise<Partial<WriteResult>>((done) => {
            lands = () => done({})
          })
        : {},
    )
    held.types('newADay', 4)
    held.settles()
    await after()

    let gone = false
    const going = flush().then(() => {
      gone = true
    })
    await after()
    expect(gone).toBe(false)
    lands()
    await going
    expect(gone).toBe(true)
  })
})

describe('a file read again', () => {
  it('leaves a setting moved since the read where the person left it', async () => {
    const { held, changed } = await opened()
    held.types('newADay', 4)
    changed(['Steady.md'])
    await after()
    expect(held.settings.value.newADay).toBe(4)
  })

  it('takes the file up where nothing stands unwritten', async () => {
    const { held, changed } = await opened()
    changed(['Steady.md'])
    await after()
    expect(held.settings.value.newADay).toBe(STEADY.newADay)
  })

  // Moving one field is not a claim on the rest: the file has the say over
  // every setting this person did not touch.
  it('takes up every setting beside the one that was moved', async () => {
    const { held, changed } = await opened({}, curve, (time) =>
      time === 0
        ? {}
        : {
            preset: {
              path: 'Steady.md',
              title: 'Steady',
              settings: { ...STEADY, newADay: 7, reviewsADay: 33, interval: 40 },
              problems: [],
              stops: StopReason.NOTHING,
              stopsOn: StopReason.NOTHING,
            },
          },
    )
    held.types('newADay', 4)
    changed(['Steady.md'])
    await after()

    expect(held.settings.value.newADay).toBe(4)
    expect(held.settings.value.reviewsADay).toBe(33)
    expect(held.settings.value.interval).toBe(40)
  })

  it('drops the problems of the file it read before, where it is refused', async () => {
    const { held, changed } = await opened({}, curve, (time) =>
      time === 0
        ? {
            preset: {
              path: 'Steady.md',
              title: 'Steady',
              settings: STEADY,
              problems: ['a line nobody could read'],
              stops: StopReason.NOTHING,
              stopsOn: StopReason.NOTHING,
            },
          }
        : { preset: null, refusal: 'notAPreset' },
    )
    expect(held.problems.value).toHaveLength(1)

    changed(['Steady.md'])
    await after()
    expect(held.problems.value).toStrictEqual([])
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

    expect(held.settings.value.minutesADay).toBe(30)
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
    expect(held.settings.value.reviewsADay).toBe(12)
    expect(held.settings.value.retention).toBe(0.95)
    expect(written.at(-1)?.reviewsADay).toBe(12)
    expect(written.at(-1)?.retention).toBe(0.95)
  })
})

// A setting the goal does not name is never deleted and never zeroed: it waits
// in the file for the goal that names it to come round again.
describe('a setting the goal on screen does not name', () => {
  it('is not drawn, and keeps its value in the file across a write', async () => {
    const { held, written } = await opened({ minutesADay: 34, newADay: 12, reviewsADay: 7 })
    expect(fieldsUnder(held.settings.value.goal, held.settings.value.learned)).not.toContain('reviewsADay')

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
    expect(held.settings.value.reviewsADay).toBe(7)
    expect(held.settings.value.newADay).toBe(12)
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
  const value = goalValue(asked, NOW)
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
    expect(held.place.value).toBe(held.curve.value.now.at)

    held.types('backlog', 5)
    held.settles()
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(held.place.value).toBe(held.curve.value.now.at)
    expect(held.settings.value.minutesADay).toBe(23)
    expect(written.at(-1)?.minutesADay).toBe(23)
  })

  const SAID: Partial<Record<Field, SettingValue>> = {
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
        const was = held.settings.value
        held.types(field, said)
        held.settles()
        await Promise.resolve()
        await Promise.resolve()
        await Promise.resolve()

        const now = held.settings.value
        expect({ ...now, [field]: was[field] }).toStrictEqual(was)
        expect(written.at(-1)).toStrictEqual(now)
      }
    }
  })
})

/**
 * A tab draws its rows before its first read lands, so a person can move one of
 * them while the file's own settings are still on their way. What they moved is
 * theirs; every other row is standing at a placeholder, and the file has the say
 * over it. Writing the placeholders back would lose the preset a person never
 * looked at.
 */
describe('a field moved before the first read lands', () => {
  it('leaves every other setting to the read, and writes what the read said', async () => {
    const { tab, written, lands } = await opening({
      retention: 0.93,
      newADay: 3,
      interval: 40,
    })
    // Nothing of the file has arrived: this is what the tab opened at.
    expect(tab.settings.value.retention).toBe(DEFAULTS.retention)
    expect(tab.settings.value.interval).toBe(DEFAULTS.interval)

    tab.types('backlog', 55)
    lands()
    await after()

    expect(tab.settings.value.backlog).toBe(55)
    expect(tab.settings.value.retention).toBe(0.93)
    expect(tab.settings.value.newADay).toBe(3)
    expect(tab.settings.value.interval).toBe(40)

    tab.settles()
    await after()
    expect(written.at(-1)).toStrictEqual(tab.settings.value)
  })

  it('keeps the moved field where the person left it, whatever the file says', async () => {
    const { tab, lands } = await opening({ backlog: 10 })
    tab.types('backlog', 55)
    lands()
    await after()
    expect(tab.settings.value.backlog).toBe(55)
  })
})

// How far a setting goes is the application's and not the window's: a field
// drawn from a copy of its own accepts a number the write then refuses.
describe('how far each setting goes', () => {
  it('is what the read answered', async () => {
    const { held } = await opened()
    expect(held.bounds.value).toStrictEqual(BOUNDS)
  })

  it('is nothing at all where the read answered none', async () => {
    const { held } = await opened({}, curve, () => ({ bounds: NO_BOUNDS }))
    expect(held.bounds.value).toStrictEqual({})
  })

  it('holds a number typed past an end at the end the read answered', async () => {
    const { held } = await opened()
    held.types('interval', 9_000)
    expect(held.settings.value.interval).toBe(BOUNDS.interval.most)

    held.types('newADay', -4)
    expect(held.settings.value.newADay).toBe(BOUNDS.newADay.least)
  })
})


// A hand on a track moves through every value between where it started and
// where it stops, and each of those is a curve nobody asked to see.
describe('a control dragged across its range', () => {
  it('keeps one curve in the air and asks again for where the hand came to rest', async () => {
    const waiting: ((one: Curve) => void)[] = []
    const { held: tab, asked } = await opened(
      { goal: 'retention' },
      () => new Promise<Curve>((take) => waiting.push(take)),
    )

    // The read's own curve is the one in the air; every step of the drag lands
    // on top of it.
    for (const share of [0.8, 0.82, 0.84, 0.86]) tab.types('retention', share)
    await after()
    expect(asked).toHaveLength(1)

    waiting[0]?.(curve)
    await after()
    expect(asked).toHaveLength(2)
  })
})
