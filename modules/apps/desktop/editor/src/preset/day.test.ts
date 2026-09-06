/**
 * The day a preset counts from is the review day the window was told.
 *
 * A day of review begins at the hour the settings name, so between midnight and
 * that hour the calendar has moved on and the review day has not. A tab
 * counting from the calendar says one number and the answer that lands says
 * another, and a date typed or written out is a day off the day the core reads.
 */
import { describe, expect, it } from 'vitest'
import { StopReason } from '@numen/protocol'
import { dayAfter } from '@numen/ui'

import { presetting } from './kind'
import { DEFAULTS, NO_BOUNDS, type Curve, type Point, type Presets, type Settings } from './core'
import { BOUNDS } from './drawn'
import type { WindowHandle } from '../tabs/windowing'
import type { FileOpeners } from '../tabs/openers'

/** The review day the window is told, which is the day holding one in the morning. */
const DAY = '2026-09-04'

/** The day the preset aims at, eight days from the one holding now. */
const BY = '2026-09-12'

/** How many places a curve of the core's is drawn at. */
const PLACES = 25

const point: Point = {
  reviews: 20,
  minutes: 10,
  retained: 0.9,
  owed: 0,
  through: 1,
  enough: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
}

/**
 * A curve of a date as the core answers one: the grid counts days from the day
 * holding now, and each place names the day it falls on.
 */
const honest = (): Curve => {
  const grid = Array.from({ length: PLACES }, (_, at) => Math.round(1 + (29 * at) / (PLACES - 1)))
  return {
    goal: 'date',
    grid,
    days: grid.map((one) => dayAfter(DAY, one)),
    at: grid.map(() => point),
    now: { at: grid.indexOf(8), value: 8, day: BY },
    suggested: { at: -1, value: 0, day: '' },
    decks: 1,
    cards: 40,
    overdue: 0,
    unbegun: 0,
    honest: true,
  }
}

/** A tab over one preset, answered with the curve given, or with none at all. */
const opened = async (settings: Partial<Settings>, answer?: Curve) => {
  const written: Settings[] = []
  const core: Presets = {
    read: async (path) => ({
      preset: {
        path,
        title: 'Sanskrit',
        settings: { ...DEFAULTS, ...settings },
        problems: [],
        stops: StopReason.NOTHING,
        stopsOn: StopReason.NOTHING,
      },
      refusal: null,
      at: 'one',
      bounds: BOUNDS,
    }),
    scheduling: async () => ({ preset: null, refusal: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    makes: async () => ({ path: '', refusal: null }),
    schedules: async () => ({ refusal: null, changed: false, at: '' }),
    write: async (_path, put) => {
      written.push(put)
      return { refusal: null, changed: false, at: 'two' }
    },
    // A curve nobody answers leaves the sketch standing, which is what the
    // arithmetic here is read off.
    curve: async () => answer ?? new Promise<Curve>(() => {}),
  }
  const handle = { closes: () => {} } as unknown as WindowHandle
  const puts = { holds: () => {} } as unknown as FileOpeners
  const kind = presetting(core, handle, puts, () => {}, () => DAY)
  const state = await kind.kind.opens('Sanskrit.md')
  for (let i = 0; i < 10; i += 1) await Promise.resolve()
  return { state, written }
}

// One in the morning: the calendar says the fifth, and the review day that
// began at four on the fourth is the day the window was told. The core counts
// eight days to the twelfth from that day, and the calendar would count seven.
describe('a goal of a date', () => {
  it('counts the days to it from the review day', async () => {
    const { state } = await opened({ goal: 'date', byDate: BY })

    expect(state.curve.value.now.value).toBe(8)
  })

  it('leaves the knob where it stands when the day it already aims at is typed', async () => {
    const answered = honest()
    const { state } = await opened({ goal: 'date', byDate: BY }, answered)
    expect(state.place.value).toBe(answered.now.at)

    state.types('byDate', BY)

    expect(state.place.value).toBe(answered.now.at)
  })

  it('opens on a day counted from the review day where the file names none', async () => {
    const { state, written } = await opened({ goal: 'minutes', byDate: '' })

    state.chooses('date')
    for (let i = 0; i < 10; i += 1) await Promise.resolve()

    expect(state.settings.value.byDate).toBe('2026-10-04')
    expect(written.at(-1)?.byDate).toBe('2026-10-04')
  })
})
