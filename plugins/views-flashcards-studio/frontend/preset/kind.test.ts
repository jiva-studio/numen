/**
 * What one preset tab holds: the knob, the fields under it, and what is
 * written.
 *
 * The goal steers one value, and the knob and the field showing it are two
 * views of it. Every other field is a person's to take out of the goal's hands.
 */
import { describe, expect, it } from 'vitest'

import { usePresetTab } from './kind'
import type { PresetTabState, SettingValue } from './types'
import { StopReason } from '../gen/numen/flashcards/v1/presets_pb.js'
import { getGoalValue, findNearest } from './lib/curve'
import { asFailure, asValue } from '@numen/wire'
import type { CardsFailure } from '../cards'
import { getGoalFields, getGoalField, type Field } from './lib/fields'
import {
  DEFAULTS,
  NO_BOUNDS,
  type Curve,
  type Goal,
  type Point,
  type Presets,
  type Preset,
  type PresetReadResult,
  type PresetWriteResult,
  type ReadPreset,
  type Settings,
} from './types'
import { BOUNDS } from './fixtures'
import { ERRORS, type ErrorCode } from '../cards/wire/errors'
import { createMockBufferHost } from '@numen/plugin-sdk/testing'
import { WORDS as words } from './words'

const point = (overrides: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  canLearnEveryCard: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...overrides,
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
  now: { index: 2, value: 20, day: '' },
  suggested: { index: 3, value: 30, day: '' },
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  isHonest: true,
}

/** The review day every test here is told, so a goal of a date counts from one place. */
const NOW = '2026-08-30'

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

/** What a test says a read answered: parts of the preset, or the rejection instead. */
type ReadOver = Partial<ReadPreset> & { readonly error?: ErrorCode }

/** What a test says a write answered: the file it left, or the rejection instead. */
type WriteOver = { readonly error?: CardsFailure; readonly at?: string }

/** A read built from what the test said of it. */
const readAs = (base: ReadPreset, overrides: ReadOver): PresetReadResult =>
  overrides.error ? asFailure(overrides.error) : asValue({ ...base, ...overrides })

/** A write built from what the test said of it. */
const writeAs = (overrides: WriteOver): PresetWriteResult =>
  overrides.error ? asFailure(overrides.error) : asValue({ at: overrides.at ?? 'two' })

const openPresetTab = async (
  settings: Partial<Settings> = {},
  answers: Curve | ((request: Settings) => Curve | Promise<Curve>) = curve,
  reading: (time: number) => ReadOver = () => ({}),
  writing: (time: number) => WriteOver | Promise<WriteOver> = () => ({}),
) => {
  const content: Settings[] = []
  const request: Goal[] = []
  const closed: string[] = []
  const messages: [string, string | undefined][] = []
  let times = 0
  let writes = 0
  const core: Presets = {
    read: async (path) =>
      readAs(
        {
          preset: {
            path,
            title: 'Steady',
            settings: { ...STEADY, ...settings },
            problems: [],
            stops: StopReason.NOTHING,
            stopsOn: StopReason.NOTHING,
          },
          at: 'one',
          bounds: BOUNDS,
        },
        reading(times++),
      ),
    getDeckPreset: async () => asValue({ preset: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    createPreset: async () => asValue({ path: '' }),
    scheduleDeck: async () => asValue({ at: '' }),
    write: async (_path, payload) => {
      content.push(payload)
      return writeAs(await writing(writes++))
    },
    curve: async (_path, params) => {
      request.push(params.goal)
      return typeof answers === 'function' ? answers(params) : answers
    },
  }
  const buffers = createMockBufferHost()
  const kind = usePresetTab({
    core,
    buffers,
    closeTab: (tab: string) => void closed.push(tab),
    writeMessage: (text, severity) => void messages.push([text, severity]),
    today: () => NOW,
  })
  const state = await kind.kind.open?.('Steady.md', 'preset:Steady.md')
  if (!state) throw new Error('card face missing')
  // The read and the curve behind it are two answers, and both are awaited.
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
  return {
    state,
    writtenFiles: content,
    requests: request,
    closed,
    messages,
    buffers,
    core,
    kind,
    getState: kind.getState,
    applyPathChanges: kind.applyPathChanges,
    flush: kind.flush,
  }
}

/** A moment for the read, the curve behind it and a write to land. */
const flushPromises = async () => {
  for (let i = 0; i < 10; i += 1) await Promise.resolve()
}

/**
 * A tab whose first read is still out. Until the test lets it land, the tab is
 * standing at the placeholder settings and not at the file's own.
 */
const opening = async (file: Partial<Settings>) => {
  let resolveRead = () => {}
  const target = new Promise<void>((settle) => {
    resolveRead = settle
  })
  const content: Settings[] = []
  const core: Presets = {
    read: async (path) => {
      await target
      return asValue({
        preset: {
          path,
          title: 'Steady',
          settings: { ...STEADY, ...file },
          problems: [],
          stops: StopReason.NOTHING,
          stopsOn: StopReason.NOTHING,
        },
        at: 'one',
        bounds: BOUNDS,
      })
    },
    getDeckPreset: async () => asValue({ preset: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    createPreset: async () => asValue({ path: '' }),
    scheduleDeck: async () => asValue({ at: '' }),
    write: async (_path, payload) => {
      content.push(payload)
      return asValue({ at: 'two' })
    },
    curve: async () => curve,
  }
  const kind = usePresetTab({
    core,
    buffers: createMockBufferHost(),
    closeTab: () => {},
    writeMessage: () => {},
    today: () => NOW,
  })
  return {
    tab: kind.kind.open?.('Steady.md', 'preset:Steady.md') as PresetTabState,
    written: content,
    resolveRead,
  }
}

describe('the value the goal steers', () => {
  it('leaves the knob, the field and what is written at one value after a drag', async () => {
    const { state, writtenFiles } = await openPresetTab()
    state.moveSlider(3)
    state.flushPending()
    await Promise.resolve()
    expect(state.place.value).toBe(3)
    expect(state.settings.value.minutesADay).toBe(30)
    expect(writtenFiles.at(-1)?.minutesADay).toBe(30)
  })

  // The knob rides 25 places of a grid and a person types whatever they like.
  // What they typed is what the file gets; the knob only shows where that
  // leaves them.
  it('is the number typed, and the knob goes to the place nearest it', async () => {
    const { state, writtenFiles } = await openPresetTab()
    state.updateSetting('minutesADay', 21)
    state.flushPending()
    await Promise.resolve()
    expect(state.place.value).toBe(2)
    expect(state.settings.value.minutesADay).toBe(21)
    expect(writtenFiles.at(-1)?.minutesADay).toBe(21)
  })

  it('is the target typed, whatever place of the grid remains nearest', async () => {
    const { state, writtenFiles } = await openPresetTab(
      { goal: 'retention' },
      {
        ...curve,
        goal: 'retention',
        grid: [0.8, 0.85, 0.9, 0.95],
        now: { index: 1, value: 0.85, day: '' },
      },
    )
    state.updateSetting('retention', 0.873)
    state.flushPending()
    await Promise.resolve()
    expect(state.place.value).toBe(1)
    expect(state.settings.value.retention).toBe(0.873)
    expect(writtenFiles.at(-1)?.retention).toBe(0.873)
  })

  it('is the day under a goal of a date, and a day typed moves the knob', async () => {
    const { state, writtenFiles } = await openPresetTab(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { index: 0, value: 10, day: '2026-09-09' } },
    )
    state.updateSetting('byDate', '2026-09-29')
    state.flushPending()
    await Promise.resolve()
    expect(state.place.value).toBe(2)
    expect(state.settings.value.byDate).toBe('2026-09-29')
    expect(writtenFiles.at(-1)?.byDate).toBe('2026-09-29')
  })

  it('is the day typed, and not the day the nearest place of the grid names', async () => {
    const { state, writtenFiles } = await openPresetTab(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { index: 0, value: 10, day: '2026-09-09' } },
    )
    state.updateSetting('byDate', '2026-09-22')
    state.flushPending()
    await Promise.resolve()
    expect(state.place.value).toBe(1)
    expect(state.settings.value.byDate).toBe('2026-09-22')
    expect(writtenFiles.at(-1)?.byDate).toBe('2026-09-22')
  })

  it('leaves a field holding no day at its current position, and the knob where it is', async () => {
    const { state } = await openPresetTab(
      { goal: 'date', byDate: '2026-09-09' },
      { ...dated, now: { index: 0, value: 10, day: '2026-09-09' } },
    )
    state.updateSetting('byDate', '')
    await Promise.resolve()
    expect(state.place.value).toBe(0)
  })
})

describe('the curve behind the knob', () => {
  it('is requested for once for the goal, and a walk of the grid asks nothing', async () => {
    const { state, requests } = await openPresetTab()
    state.moveSlider(1)
    state.moveSlider(3)
    state.flushPending()
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()
    expect(requests).toStrictEqual(['minutes'])
  })

  it('is requested for again where a field the knob does not ride is typed', async () => {
    const { state, requests } = await openPresetTab()
    state.updateSetting('newADay', 4)
    await Promise.resolve()
    expect(requests).toStrictEqual(['minutes', 'minutes'])
  })

  // What the material evaluates to is a fact about the vault: no setting moves one
  // of those figures, so they are kept beside the curve and stand at what the
  // last answer counted them while the next one is out.
  it('keeps what the material evaluates to while a curve is on its way', async () => {
    const { state } = await openPresetTab()
    expect(state.material.value).toStrictEqual({ decks: 1, cards: 400, overdue: 0, unbegun: 0 })

    state.updateSetting('newADay', 4)
    expect(state.isWaiting.value).toBe(true)
    expect(state.curve.value.isHonest).toBe(false)
    expect(state.material.value).toStrictEqual({ decks: 1, cards: 400, overdue: 0, unbegun: 0 })
  })

  it('counts the material at nothing until an answer has counted it', async () => {
    const { state } = await openPresetTab({}, () => new Promise<Curve>(() => {}))
    expect(state.material.value).toBeNull()
  })

  // A goal already worked out is drawn again as it was, so moving between the
  // three is instant and never puts the picture back into its waiting state.
  it('is requested for once for each goal, however often they are moved between', async () => {
    const { state, requests } = await openPresetTab()
    const settle = async () => {
      for (let i = 0; i < 4; i += 1) await Promise.resolve()
    }

    state.chooseGoal('retention')
    await settle()
    state.chooseGoal('date')
    await settle()
    expect(requests).toStrictEqual(['minutes', 'retention', 'date'])

    state.chooseGoal('minutes')
    await settle()
    expect(state.curve.value.isHonest).toBe(true)
    state.chooseGoal('retention')
    await settle()
    expect(state.curve.value.isHonest).toBe(true)
    expect(requests).toStrictEqual(['minutes', 'retention', 'date'])
  })

  it('leaves the knob at its current position where the file says what it already returned', async () => {
    const { state, applyPathChanges } = await openPresetTab()
    state.moveSlider(3)
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.place.value).toBe(3)
  })

  // The counts behind a curve are the vault's, not the preset's, so a file read
  // again is a curve to ask for again however little the settings moved.
  it('is requested afresh on a re-read, so a deck pointed here since is seen', async () => {
    let decks = 0
    const { state, applyPathChanges } = await openPresetTab({}, () => ({
      ...curve,
      decks: decks++,
    }))
    expect(state.curve.value.decks).toBe(0)

    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.curve.value.decks).toBe(1)
  })

  // The range the line is drawn over runs to the value the knob rides, so a
  // value typed past the end of it is a curve nobody has been answered.
  it('is requested for again where a value past the end of the range is typed', async () => {
    const createReachingCurve = (settings: Settings): Curve => ({
      ...curve,
      grid: [0, settings.minutesADay / 2, settings.minutesADay],
      at: [point(), point(), point()],
      now: { index: 2, value: settings.minutesADay, day: '' },
    })
    const { state, requests } = await openPresetTab({}, createReachingCurve)
    expect(state.curve.value.grid.at(-1)).toBe(20)

    state.updateSetting('minutesADay', 120)
    await flushPromises()
    expect(requests).toStrictEqual(['minutes', 'minutes'])
    expect(state.curve.value.grid.at(-1)).toBe(120)
  })

  // The figures over the picture are read as the answer to what relies on
  // screen, so the run of a settled question is nobody's answer to a new one.
  it('is nobody’s answer while the answer to the settings now standing is out', async () => {
    const { state } = await openPresetTab()
    expect(state.curve.value.isHonest).toBe(true)

    state.updateSetting('newADay', 4)
    expect(state.curve.value.isHonest).toBe(false)
    await flushPromises()
    expect(state.curve.value.isHonest).toBe(true)
  })
})

// The goal is the one choice a person makes outright, and it is theirs the
// moment they make it. A window shut before an answer lands loses nothing.
describe('the goal chosen', () => {
  it('is written without waiting on the curve behind it', async () => {
    const { state, writtenFiles } = await openPresetTab({}, () => new Promise<Curve>(() => {}))
    state.chooseGoal('retention')
    await flushPromises()
    expect(writtenFiles.at(-1)?.goal).toBe('retention')
  })

  it('names a day where the file names none, since a date is aimed at one', async () => {
    const { state, writtenFiles } = await openPresetTab({ byDate: '' }, dated)
    state.chooseGoal('date')
    await flushPromises()
    expect(state.settings.value.byDate).toBe('2026-09-29')
    expect(writtenFiles.at(-1)?.byDate).toBe('2026-09-29')
  })

  it('keeps the day the file names, whether it is ahead of today or behind', async () => {
    for (const day of ['2026-12-25', '2026-08-30', '2026-01-06']) {
      const { state, writtenFiles } = await openPresetTab({ byDate: day }, dated)
      state.chooseGoal('date')
      await flushPromises()
      expect(state.settings.value.byDate).toBe(day)
      expect(writtenFiles.at(-1)?.byDate).toBe(day)
    }
  })
})

// A tab is the one thing that holds a preset. Anything else answering for one
// answers with settings nobody set, and writes them into the file at the first
// control release.
describe('a preset no tab has open', () => {
  it('is retained by nothing, where no tab ever opened it', async () => {
    const { getState } = await openPresetTab()
    expect(getState('Nowhere.md')).toBeUndefined()
  })

  it('is what a preset becomes once its tab is shut', async () => {
    const { state, getState } = await openPresetTab()
    state.close('Steady.md')
    await flushPromises()
    expect(getState('Steady.md')).toBeUndefined()
  })

  it('is what a renamed preset becomes once its tab is shut', async () => {
    const { state, getState, applyPathChanges } = await openPresetTab()
    applyPathChanges([], [{ from: 'Steady.md', to: 'Slow.md' }])
    state.close('Slow.md')
    await flushPromises()
    expect(getState('Slow.md')).toBeUndefined()
    expect(getState('Steady.md')).toBeUndefined()
  })
})

// A person who is told what happened can do something about it. One sentence
// over every error names none of them.
describe('what the tab says it encountered as an error', () => {
  const errors: readonly ErrorCode[] = ['missing', 'tooLarge', 'notANote', 'unreadable']

  it('is a sentence of its own for each error a read answers', async () => {
    const messages: string[] = []
    for (const error of errors) {
      const { state } = await openPresetTab({}, curve, () => ({ error }))
      messages.push(state.errorMessage.value)
    }
    expect(messages.every((one) => one !== '')).toBe(true)
    expect(new Set(messages).size).toBe(errors.length)
  })

  it('is a sentence of its own for each error a write answers', async () => {
    const messages: string[] = []
    for (const error of errors) {
      const { state } = await openPresetTab(
        {},
        curve,
        () => ({}),
        () => ({ error }),
      )
      state.updateSetting('newADay', 4)
      state.flushPending()
      await flushPromises()
      messages.push(state.errorMessage.value)
    }
    expect(messages.every((one) => one !== '')).toBe(true)
    expect(new Set(messages).size).toBe(errors.length)
  })

  // The settings are in the tab and nowhere else once a write fails, and
  // a person deciding what to do next has to be told that.
  it('says the settings are still here where a write failed', async () => {
    for (const error of errors) {
      expect(words.notSaved(error)).toContain('are kept')
    }
  })

  it('says the error of a read and the error of a write in different words', async () => {
    for (const error of errors) {
      expect(words.notRead(error)).not.toBe(words.notSaved(error))
    }
  })
})

// A picture that says it is indexing the vault says an answer is on its way.
// Where none is coming, the tab says what happened and stops saying it.
describe('a curve nobody answers', () => {
  it('leaves the tab saying why, and not saying it is reading', async () => {
    const { state } = await openPresetTab({}, () => Promise.reject(new Error('the vault is gone')))
    expect(state.isWaiting.value).toBe(false)
    expect(state.errorMessage.value).not.toBe('')
  })

  it('is what a file rejected leaves, so no answer is waited on', async () => {
    const { state } = await openPresetTab({}, curve, () => ({ error: 'notAPreset' }))
    expect(state.isWaiting.value).toBe(false)
    expect(state.errorMessage.value).not.toBe('')
  })

  it('reschedules review when target goal has no reviews', async () => {
    const { state } = await openPresetTab({}, () => new Promise<Curve>(() => {}))
    state.chooseGoal('retention')
    await flushPromises()
    expect(state.isWaiting.value).toBe(true)
  })
})

// A setting typed is written when the control is release, so at any moment
// where it is asked to go, and where the window is.
describe('what a tab still owes the file', () => {
  it('is written before the tab goes', async () => {
    const { state, writtenFiles, closed } = await openPresetTab()
    state.updateSetting('newADay', 4)
    state.close('Steady.md')
    await flushPromises()
    expect(writtenFiles.at(-1)?.newADay).toBe(4)
    expect(closed).toStrictEqual(['Steady.md'])
  })

  it('keeps the tab open where the write failed, and says why', async () => {
    const { state, closed } = await openPresetTab(
      {},
      curve,
      () => ({}),
      () => ({ error: 'notAPreset' }),
    )
    state.updateSetting('newADay', 4)
    state.close('Steady.md')
    await flushPromises()
    expect(closed).toStrictEqual([])
    expect(state.errorMessage.value).not.toBe('')
  })

  it('lets the tab go the second time it is requested, the person having been told', async () => {
    const { state, closed } = await openPresetTab(
      {},
      curve,
      () => ({}),
      () => ({ error: 'notAPreset' }),
    )
    state.updateSetting('newADay', 4)
    state.close('Steady.md')
    await flushPromises()
    state.close('Steady.md')
    await flushPromises()
    expect(closed).toStrictEqual(['Steady.md'])
  })

  it('keeps the tab open where the file moved under it and nothing was written', async () => {
    const { state, closed } = await openPresetTab(
      {},
      curve,
      () => ({}),
      () => ({ error: 'changed' }),
    )
    state.updateSetting('newADay', 4)
    state.close('Steady.md')
    await flushPromises()
    expect(closed).toStrictEqual([])
    expect(state.hasExternalChange.value).toBe(true)
  })

  it('is written when the window goes', async () => {
    const { state, writtenFiles, flush } = await openPresetTab()
    state.updateSetting('newADay', 4)
    await flush()
    expect(writtenFiles.at(-1)?.newADay).toBe(4)
  })

  it('is waited for by the window going, where a write is already out', async () => {
    let resolveWrite = () => {}
    const { state, flush } = await openPresetTab(
      {},
      curve,
      () => ({}),
      (time) =>
        time === 0
          ? new Promise<WriteOver>((isDone) => {
              resolveWrite = () => isDone({})
            })
          : {},
    )
    state.updateSetting('newADay', 4)
    state.flushPending()
    await flushPromises()

    let isDone = false
    const going = flush().then(() => {
      isDone = true
    })
    await flushPromises()
    expect(isDone).toBe(false)
    resolveWrite()
    await going
    expect(isDone).toBe(true)
  })
})

describe('a file read again', () => {
  it('leaves a setting moved since the read where the person left it', async () => {
    const { state, applyPathChanges } = await openPresetTab()
    state.updateSetting('newADay', 4)
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.newADay).toBe(4)
  })

  it('takes the file up where nothing remains unwritten', async () => {
    const { state, applyPathChanges } = await openPresetTab()
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.newADay).toBe(STEADY.newADay)
  })

  // Moving one field is not a claim on the rest: the file has the say over
  // every setting this person did not touch.
  it('takes up every setting alongside the one that was moved', async () => {
    const { state, applyPathChanges } = await openPresetTab({}, curve, (time) =>
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
    state.updateSetting('newADay', 4)
    applyPathChanges(['Steady.md'])
    await flushPromises()

    expect(state.settings.value.newADay).toBe(4)
    expect(state.settings.value.reviewsADay).toBe(33)
    expect(state.settings.value.interval).toBe(40)
  })

  it('drops the problems of the file it read before, where it fails', async () => {
    const { state, applyPathChanges } = await openPresetTab({}, curve, (time) =>
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
        : { error: 'notAPreset' },
    )
    expect(state.problems.value).toHaveLength(1)

    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.problems.value).toStrictEqual([])
  })
})

// The goal names one budget and moves that. Every other setting is the
// person's, and the control neither predicts it nor writes it.
describe('what the control writes', () => {
  it('is the minutes alone under a goal of minutes, whatever the counts hold', async () => {
    // The case reported: a day of minutes the card limits had been cut to
    // nothing under.
    const { state, writtenFiles } = await openPresetTab({
      minutesADay: 34,
      newADay: 12,
      reviewsADay: 0,
    })
    state.moveSlider(3)
    state.flushPending()
    await Promise.resolve()

    expect(state.settings.value.minutesADay).toBe(30)
    expect(writtenFiles.at(-1)?.minutesADay).toBe(30)
    expect(writtenFiles.at(-1)?.newADay).toBe(12)
    expect(writtenFiles.at(-1)?.reviewsADay).toBe(0)
    expect(writtenFiles.at(-1)?.retention).toBe(0.88)
  })

  it('leaves the counts and the target alone under a goal of minutes', async () => {
    const { state, writtenFiles } = await openPresetTab({ reviewsADay: 12, retention: 0.95 })
    state.moveSlider(1)
    state.flushPending()
    await Promise.resolve()
    expect(state.settings.value.reviewsADay).toBe(12)
    expect(state.settings.value.retention).toBe(0.95)
    expect(writtenFiles.at(-1)?.reviewsADay).toBe(12)
    expect(writtenFiles.at(-1)?.retention).toBe(0.95)
  })
})

// A setting the goal does not name is never deleted and never zeroed: it waits
// in the file for the goal that names it to come round again.
describe('a setting the goal on screen does not name', () => {
  it('is not drawn, and keeps its value in the file across a write', async () => {
    const { state, writtenFiles } = await openPresetTab({
      minutesADay: 34,
      newADay: 12,
      reviewsADay: 7,
    })
    expect(getGoalFields(state.settings.value.goal, state.settings.value.learned)).not.toContain(
      'reviewsADay',
    )

    state.moveSlider(1)
    state.flushPending()
    await Promise.resolve()
    expect(writtenFiles.at(-1)?.reviewsADay).toBe(7)
    expect(writtenFiles.at(-1)?.newADay).toBe(12)
  })

  it('arrives back the moment its own goal is chosen again', async () => {
    const { state, writtenFiles } = await openPresetTab({
      minutesADay: 34,
      newADay: 12,
      reviewsADay: 7,
    })
    state.moveSlider(3)
    state.flushPending()
    await Promise.resolve()

    state.chooseGoal('retention')
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(getGoalFields('retention', DEFAULTS.learned)).toContain('reviewsADay')
    expect(state.settings.value.reviewsADay).toBe(7)
    expect(state.settings.value.newADay).toBe(12)
    expect(writtenFiles.at(-1)?.reviewsADay).toBe(7)
  })
})

/**
 * A curve worked out off the settings it was asked under, the way the
 * application works one out: its range runs to what carrying the load costs,
 * which the share of the day going to the debt moves.
 */
const createRangedCurve = (settings: Settings): Curve => {
  const top = 60 + settings.backlog
  const grid = Array.from({ length: 7 }, (_, index) => Math.round((top * (index + 1)) / 7))
  const value = getGoalValue(settings, NOW)
  return {
    ...curve,
    goal: settings.goal,
    grid,
    at: grid.map((minutes) => point({ minutes, reviews: minutes * 4 })),
    now: { index: findNearest(grid, value), value, day: '' },
    suggested: { index: grid.length - 1, value: grid[grid.length - 1] ?? 0, day: '' },
  }
}

// Nothing but the field a person types in may change, and the budget the goal
// steers least of all: it is what the picture is scaled to.
describe('a field the goal does not getGoalField, typed', () => {
  // A curve asked for under a new share of the day comes back over a range of
  // knob goes back to where the preset stands, so the picture and the file say
  // one thing and the next touch of the knob writes what the file already says.
  it('leaves the knob standing where the preset remains, whatever range arrives back', async () => {
    const { state, writtenFiles } = await openPresetTab({ minutesADay: 23 }, createRangedCurve)
    expect(state.place.value).toBe(state.curve.value.now.index)

    state.updateSetting('backlog', 5)
    state.flushPending()
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(state.place.value).toBe(state.curve.value.now.index)
    expect(state.settings.value.minutesADay).toBe(23)
    expect(writtenFiles.at(-1)?.minutesADay).toBe(23)
  })

  const NEW_VALUES: Partial<Record<Field, SettingValue>> = {
    newADay: 4,
    reviewsADay: 33,
    counts: 'shows',
    backlog: 5,
    load: { sat: 50 },
    hasEvenLoad: false,
  }

  it('leaves every other setting exactly as it stood, under every goal', async () => {
    for (const goal of ['minutes', 'retention'] as const) {
      for (const field of getGoalFields(goal, DEFAULTS.learned)) {
        const value = NEW_VALUES[field]
        if (field === getGoalField(goal) || value === undefined) continue

        const { state, writtenFiles } = await openPresetTab(
          { goal, minutesADay: 23 },
          createRangedCurve,
        )
        const was = state.settings.value
        state.updateSetting(field, value)
        state.flushPending()
        await Promise.resolve()
        await Promise.resolve()
        await Promise.resolve()

        const now = state.settings.value
        expect({ ...now, [field]: was[field] }).toStrictEqual(was)
        expect(writtenFiles.at(-1)).toStrictEqual(now)
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
  it('leaves every other setting to the read, and writes what the read returned', async () => {
    const {
      tab,
      written: writtenFiles,
      resolveRead,
    } = await opening({
      retention: 0.93,
      newADay: 3,
      interval: 40,
    })
    // Nothing of the file has arrived: this is what the tab opened at.
    expect(tab.settings.value.retention).toBe(DEFAULTS.retention)
    expect(tab.settings.value.interval).toBe(DEFAULTS.interval)

    tab.updateSetting('backlog', 55)
    resolveRead()
    await flushPromises()

    expect(tab.settings.value.backlog).toBe(55)
    expect(tab.settings.value.retention).toBe(0.93)
    expect(tab.settings.value.newADay).toBe(3)
    expect(tab.settings.value.interval).toBe(40)

    tab.flushPending()
    await flushPromises()
    expect(writtenFiles.at(-1)).toStrictEqual(tab.settings.value)
  })

  it('keeps the moved field where the person left it, whatever the file says', async () => {
    const { tab, resolveRead } = await opening({ backlog: 10 })
    tab.updateSetting('backlog', 55)
    resolveRead()
    await flushPromises()
    expect(tab.settings.value.backlog).toBe(55)
  })
})

// How far a setting goes is the application's and not the window's: a field
// drawn from a copy of its own accepts a number the write then refuses.
describe('how far each setting goes', () => {
  it('is what the read answered', async () => {
    const { state } = await openPresetTab()
    expect(state.bounds.value).toStrictEqual(BOUNDS)
  })

  it('is nothing at all where the read answered none', async () => {
    const { state } = await openPresetTab({}, curve, () => ({ bounds: NO_BOUNDS }))
    expect(state.bounds.value).toStrictEqual({})
  })

  it('holds a number typed past an end at the end the read answered', async () => {
    const { state } = await openPresetTab()
    state.updateSetting('interval', 9_000)
    expect(state.settings.value.interval).toBe(BOUNDS.interval.most)

    state.updateSetting('newADay', -4)
    expect(state.settings.value.newADay).toBe(BOUNDS.newADay.least)
  })

  it('holds each day of the week inside the share the read answered', async () => {
    const { state } = await openPresetTab()
    state.updateSetting('load', { sat: 250, sun: -10, mon: 50 })
    expect(state.settings.value.load).toStrictEqual({
      sat: BOUNDS.load.most,
      sun: BOUNDS.load.least,
      mon: 50,
    })
  })
})

// A hand on a track moves through every value between where it started and
// where it stops, and each of those is a curve nobody asked to see.
describe('a control dragged across its range', () => {
  it('keeps one curve in the air and asks again for where the hand came to rest', async () => {
    const waiters: ((one: Curve) => void)[] = []
    const { state: tab, requests } = await openPresetTab(
      { goal: 'retention' },
      () => new Promise<Curve>((take) => waiters.push(take)),
    )

    // The read's own curve is the one in the air; every step of the drag lands
    // on top of it.
    for (const share of [0.8, 0.82, 0.84, 0.86]) tab.updateSetting('retention', share)
    await flushPromises()
    expect(requests).toHaveLength(1)

    waiters[0]?.(curve)
    await flushPromises()
    expect(requests).toHaveLength(2)
  })
})

const createPresetRead = (settings: Settings, title = 'Steady'): ReadOver => ({
  preset: {
    path: 'Steady.md',
    title,
    settings,
    problems: [],
    stops: StopReason.NOTHING,
    stopsOn: StopReason.NOTHING,
  },
})

/** A write that never comes back, so what the tab owes stays owed. */
const neverWritten = () => new Promise<WriteOver>(() => {})

describe('the tab a preset is drawn in', () => {
  it('is the state of the open preset, requested for by its path', async () => {
    const { state, getState } = await openPresetTab()
    expect(getState('Steady.md')?.id).toBe('Steady.md')
    expect(getState('Steady.md')?.settings).toBe(state.settings)
  })

  it('is called by the title the file names, and found again by its path', async () => {
    const { state, kind } = await openPresetTab()
    expect(kind.kind.getTitle?.(state)).toBe('Steady')
    expect(kind.kind.identity?.('Steady.md')).toBe('Steady.md')
    expect(kind.kind.getTarget?.(state)).toStrictEqual({ path: 'Steady.md', title: 'Steady' })
    expect(kind.kind.getOpenTab?.(state)).toStrictEqual({ path: 'Steady.md' })
  })

  it('is called by its file where no title is known, and by a word where no file is', async () => {
    const { kind } = await openPresetTab()
    expect(kind.getTitle('presets/Plan.md')).toBe('Plan.md')
    expect(kind.getTitle('')).toBe(words.newPreset)
  })

  it('keeps the title it was read under where a later read names none', async () => {
    const { state, kind } = await openPresetTab({}, curve, (time) =>
      time === 0 ? {} : createPresetRead(STEADY, ''),
    )
    state.reload()
    await flushPromises()
    expect(kind.getTitle('Steady.md')).toBe('Steady')
  })
})

describe('a preset moved to another path', () => {
  it('takes its title and its tab to the new path, and leaves nothing at the old one', async () => {
    const { state, kind, getState, applyPathChanges } = await openPresetTab()
    applyPathChanges([], [{ from: 'Steady.md', to: 'Slow.md' }])
    expect(kind.getTitle('Slow.md')).toBe('Steady')
    expect(kind.getTitle('Steady.md')).toBe('Steady.md')
    expect(getState('Slow.md')?.settings).toBe(state.settings)
    expect(getState('Steady.md')).toBeUndefined()
  })

  it('moves nothing where no tab holds the path', async () => {
    const reads: number[] = []
    const { applyPathChanges } = await openPresetTab({}, curve, (time) => {
      reads.push(time)
      return {}
    })
    expect(() => applyPathChanges([], [{ from: 'Other.md', to: 'Else.md' }])).not.toThrow()
    applyPathChanges(['Other.md'])
    await flushPromises()
    expect(reads).toStrictEqual([0])
  })
})

describe('a tab before its first read lands', () => {
  it('holds no problems', async () => {
    const { tab } = await opening({})
    expect(tab.problems.value).toStrictEqual([])
  })
})

describe('a tab read again', () => {
  it('reads the file again when requested to', async () => {
    const { state } = await openPresetTab({}, curve, (time) =>
      time === 0 ? {} : createPresetRead({ ...STEADY, newADay: 7 }),
    )
    state.reload()
    await flushPromises()
    expect(state.settings.value.newADay).toBe(7)
  })

  it('stops saying a read failed once a read succeeds', async () => {
    const { state } = await openPresetTab({}, curve, (time) =>
      time === 0 ? { error: 'missing' } : {},
    )
    expect(state.errorMessage.value).not.toBe('')
    state.reload()
    await flushPromises()
    expect(state.errorMessage.value).toBe('')
  })
})

// What a person chose and has not seen written is theirs, and a read of the
// file does not take it back.
describe('a choice not yet written, across a read', () => {
  it('keeps the goal chosen', async () => {
    const { state, applyPathChanges } = await openPresetTab({}, curve, () => ({}), neverWritten)
    state.chooseGoal('retention')
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.goal).toBe('retention')
  })

  it('keeps the day a date goal named where the file named none', async () => {
    const { state, applyPathChanges } = await openPresetTab(
      { byDate: '' },
      dated,
      () => ({}),
      neverWritten,
    )
    state.chooseGoal('date')
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.byDate).toBe('2026-09-29')
  })

  it('takes up the file’s day where the goal chosen kept the day it had', async () => {
    const { state, applyPathChanges } = await openPresetTab(
      { byDate: '2026-12-25' },
      dated,
      (time) => (time === 0 ? {} : createPresetRead({ ...STEADY, byDate: '2027-01-01' })),
      neverWritten,
    )
    state.chooseGoal('date')
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.goal).toBe('date')
    expect(state.settings.value.byDate).toBe('2027-01-01')
  })

  it('keeps where the knob was moved to', async () => {
    const { state, applyPathChanges } = await openPresetTab()
    state.moveSlider(3)
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.minutesADay).toBe(30)
  })

  it('takes up the file’s value where what was typed changed nothing', async () => {
    const { state, applyPathChanges } = await openPresetTab({}, curve, (time) =>
      time === 0 ? {} : createPresetRead({ ...STEADY, counts: 'shows' }),
    )
    state.updateSetting('counts', 'hours')
    applyPathChanges(['Steady.md'])
    await flushPromises()
    expect(state.settings.value.counts).toBe('shows')
  })
})

describe('the goal chosen again', () => {
  it('writes nothing where it is the goal already standing', async () => {
    const { state, writtenFiles } = await openPresetTab()
    state.chooseGoal('minutes')
    await flushPromises()
    expect(writtenFiles).toStrictEqual([])
  })
})

describe('the knob, as a field is typed', () => {
  it('stays at its current position where the field is not the one the goal steers', async () => {
    const { state } = await openPresetTab()
    state.updateSetting('newADay', 4)
    await flushPromises()
    expect(state.place.value).toBe(2)
  })

  it('goes to the first place where the value typed is nearest it', async () => {
    const { state } = await openPresetTab()
    state.updateSetting('minutesADay', 1)
    await flushPromises()
    expect(state.place.value).toBe(0)
  })

  it('goes to the place nearest a value typed that the curve already answers', async () => {
    const { state, requests } = await openPresetTab(
      {},
      { ...curve, now: { index: 1, value: 20, day: '' } },
    )
    expect(state.place.value).toBe(1)
    state.updateSetting('minutesADay', 20)
    await flushPromises()
    expect(state.place.value).toBe(2)
    expect(requests).toStrictEqual(['minutes'])
  })
})

describe('the preset of a note', () => {
  const deck = (preset: Preset | null, error?: ErrorCode) => async () =>
    error ? asFailure(error) : asValue({ preset, at: '', bounds: NO_BOUNDS })
  const note = (path: string, type: string) => ({ path, title: '', type })
  const SLOW: Preset = {
    path: 'presets/Slow.md',
    title: 'Slow',
    settings: STEADY,
    problems: [],
    stops: StopReason.NOTHING,
    stopsOn: StopReason.NOTHING,
  }

  it('is a preset itself, and a note too', async () => {
    const { kind } = await openPresetTab()

    expect(await kind.findPreset(note('Plan.md', 'preset'))).toStrictEqual({
      ok: true,
      at: 'Plan.md',
    })
    expect(await kind.findPreset(note('Plan.md', 'note'))).toStrictEqual({
      ok: true,
      at: 'Plan.md',
    })
  })

  it('of a deck is the preset it is scheduled by, called what that preset is called', async () => {
    const { kind, core } = await openPresetTab()
    core.getDeckPreset = deck(SLOW)

    expect(await kind.findPreset(note('Animals.md', 'deck'))).toStrictEqual({
      ok: true,
      at: 'presets/Slow.md',
    })
    expect(kind.getTitle('presets/Slow.md')).toBe('Slow')
  })

  it('of a deck naming no preset is none, and the defaults schedule it', async () => {
    const { kind, core } = await openPresetTab()
    core.getDeckPreset = deck({ ...SLOW, path: '', title: '' })

    expect(await kind.findPreset(note('Animals.md', 'deck'))).toStrictEqual({
      ok: false,
      text: words.noPreset,
      severity: 'caution',
    })
  })

  it('of a deck the vault would not read is none, with the reason', async () => {
    const { kind, core } = await openPresetTab()
    core.getDeckPreset = deck(null, 'notADeck')

    expect(await kind.findPreset(note('Animals.md', 'deck'))).toStrictEqual({
      ok: false,
      text: ERRORS.notADeck,
      severity: 'error',
    })
  })

  it('opens in a preset tab where there is one', async () => {
    const { kind, core, messages } = await openPresetTab()
    core.getDeckPreset = deck(SLOW)
    const openedTabs: [string, string][] = []

    await kind.openPresetOf(
      note('Animals.md', 'deck'),
      (one, path) => void openedTabs.push([one, path]),
    )

    expect(openedTabs).toStrictEqual([['preset', 'presets/Slow.md']])
    expect(messages).toStrictEqual([])
  })

  it('writes why there is none and opens nothing', async () => {
    const { kind, core, messages } = await openPresetTab()
    core.getDeckPreset = deck({ ...SLOW, path: '', title: '' })
    const openedTabs: [string, string][] = []

    await kind.openPresetOf(
      note('Animals.md', 'deck'),
      (one, path) => void openedTabs.push([one, path]),
    )

    expect(openedTabs).toStrictEqual([])
    expect(messages).toStrictEqual([[words.noPreset, 'caution']])
  })
})

describe('a preset made', () => {
  it('is requested of the vault under the name and the folder it was given', async () => {
    const { kind, core } = await openPresetTab()
    core.createPreset = async (title, folder) => asValue({ path: `${folder}/${title}.md` })

    expect(await kind.kind.createFile?.('plans', 'Slow')).toStrictEqual(
      asValue({ path: 'plans/Slow.md' }),
    )
  })
})

describe('the presets written out when the window quits', () => {
  it('are the ones the tabs owe, written by the buffers of the window', async () => {
    const { state, writtenFiles, buffers } = await openPresetTab()
    state.updateSetting('newADay', 4)

    await buffers.flush()

    expect(writtenFiles.at(-1)?.newADay).toBe(4)
  })
})

describe('what the window asks of a preset tab', () => {
  it('is to write what it owes out, and answer once it is', async () => {
    const { state, writtenFiles, kind } = await openPresetTab()
    state.updateSetting('newADay', 4)

    await kind.kind.flushPending?.(state)

    expect(writtenFiles.at(-1)?.newADay).toBe(4)
  })

  it('is to write what a preset owes out wherever its file went since the tab opened', async () => {
    const { state, writtenFiles, kind } = await openPresetTab()
    state.updateSetting('newADay', 4)
    kind.applyPathChanges(['Beside.md'], [{ from: 'Steady.md', to: 'Beside.md' }])

    await kind.kind.flushPending?.(state)

    expect(writtenFiles.at(-1)?.newADay).toBe(4)
  })

  it('is to write nothing for a tab it does not hold', async () => {
    const { writtenFiles, kind } = await openPresetTab()

    await kind.kind.flushPending?.({ id: 'Elsewhere.md' } as never)

    expect(writtenFiles).toStrictEqual([])
  })
})

describe('the tab closed by the window', () => {
  it('is left to the preset, which writes what it owes and then closes it', async () => {
    const { state, kind, closed } = await openPresetTab()
    expect(kind.kind.onClose?.(state, 'Steady.md')).toBe(false)
    await flushPromises()
    expect(closed).toStrictEqual(['Steady.md'])
  })
})

describe('a vault that does not answer', () => {
  it('leaves the tab saying so, and not saying it is reading', async () => {
    const { state } = await openPresetTab({}, curve, () => {
      throw new Error('the vault is gone')
    })
    expect(state.errorMessage.value).toBe(words.unreachable)
    expect(state.isWaiting.value).toBe(false)
  })

  it('says so where the curve of a goal chosen is not answered', async () => {
    const { state } = await openPresetTab(
      {},
      (request) => (request.goal === 'retention' ? Promise.reject(new Error('gone')) : curve),
      () => ({}),
      neverWritten,
    )
    state.chooseGoal('retention')
    await flushPromises()
    expect(state.errorMessage.value).toBe(words.noCurve)
  })

  it('says so where the curve of a setting typed is not answered', async () => {
    const { state } = await openPresetTab({}, (request) =>
      request.newADay === 4 ? Promise.reject(new Error('gone')) : curve,
    )
    state.updateSetting('newADay', 4)
    await flushPromises()
    expect(state.errorMessage.value).toBe(words.noCurve)
  })
})
