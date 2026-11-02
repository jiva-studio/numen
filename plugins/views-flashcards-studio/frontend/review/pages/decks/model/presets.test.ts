import { describe, expect, it } from 'vitest'
import { ErrorCode } from '../../../../api/types.js'
import { StopReason } from '../../../../api/types.js'
import { DEFAULT_SETTINGS, formatSettings } from '../../../../cards/wire/settings'
import type { SettingsMessage } from '../../../../cards/wire/settings'

import { useVaultPresets } from './presets'
import { PRESET_UNREADABLE_MESSAGE } from '../api/presets'
import type { PresetsClient } from '../api/presets'
import { canStart, getSpentShare, isSpent } from '../lib/progress'
import { CLOSES_NOTHING } from '../types'
import type { Budget, Preset, Settings } from '../types'
import { getGoalWords, getLeftWords, getStoppedWords, STOPPED } from '../words'
import type { BudgetKeys, PresetCardsDue, VaultCardsDue } from '../../../entities/vault'

const settings = (fields: Partial<Settings> = {}): Settings => ({
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  load: {},
  hasEvenLoad: true,
  ...fields,
})

/** The same settings, as the schema carries them. */
const createSettingsMessage = (one: Settings): SettingsMessage =>
  formatSettings({ ...DEFAULT_SETTINGS, ...one })

const budget = (fields: Partial<Budget> = {}): Budget => ({
  new: 10,
  reviews: 200,
  minutes: 20,
  ...fields,
})

/** A preset steered by how long its day runs, which is the ordinary one. */
const byMinutes: BudgetKeys = { new: '', reviews: '', minutes: 'minutes_a_day' }

/** A preset steered by counts of cards, where the counts close the day. */
const byCounts: BudgetKeys = { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' }

const preset = (fields: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  reviewCards: 30,
  cards: 30,
  budget: budget(),
  closes: byMinutes,
  reviewed: 0,
  reviewedNew: 0,
  reviewedReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...fields,
})

/** One preset of a vault as returned by the count. */
const presetDue = (fields: Partial<PresetCardsDue> = {}): PresetCardsDue => ({
  preset: 'Sanskrit.md',
  title: 'Sanskrit',
  decks: 1,
  cards: 30,
  pending: 0,
  reviewed: 0,
  reviewedNew: 0,
  reviewedReviews: 0,
  took: 0,
  new: 10,
  reviews: 200,
  minutes: 20,
  closes: byMinutes,
  stopsOn: StopReason.NOTHING,
  ...fields,
})

const vault = (
  decks: readonly { deck: string; due: number; new: number }[],
  presets: readonly PresetCardsDue[] = [],
): VaultCardsDue => ({
  vault: '01A',
  name: 'Vault',
  path: '/vaults/01A',
  isCounted: true,
  reviewCards: 0,
  due: 0,
  new: 0,
  decks: decks.map((one) => ({
    deck: one.deck,
    reviewCards: 0,
    due: one.due,
    new: one.new,
    learned: 0,
    unbegun: 0,
  })),
  presets,
  unread: '',
  isIndexing: false,
})

/** An application answering one preset for each deck named here. */
const createPresets = (
  by: Record<string, { path: string; title: string; settings: Settings; stopsOn?: StopReason }>,
): PresetsClient => ({
  async getVaultDeckPreset({ deck }) {
    const one = by[deck]
    if (!one) return {}
    return {
      preset: {
        path: one.path,
        title: one.title,
        settings: createSettingsMessage(one.settings),
        problems: [],
        stopsOn: one.stopsOn ?? StopReason.NOTHING,
      },
    }
  },
})

// Why a preset schedules nothing is the core's verdict, rendered here in words.
// Two of the reasons name a day: the day the goal targets, which the settings
// carry, and the current day of the week.
describe('why a preset schedules nothing, in words', () => {
  const on = '2026-09-05'

  it('says nothing at all of a preset that schedules', () => {
    expect(getStoppedWords(StopReason.NOTHING, settings(), on)).toBe('')
    // A build that reports nothing is read as scheduling.
    expect(getStoppedWords(StopReason.UNSPECIFIED, settings(), on)).toBe('')
  })

  it('says which budget remains at nothing', () => {
    expect(getStoppedWords(StopReason.NO_MINUTES, settings(), on)).toBe('no time budget')
    expect(getStoppedWords(StopReason.NO_CARDS, settings(), on)).toBe('no cards per day')
    expect(getStoppedWords(StopReason.NO_DAY, settings(), on)).toBe('no target date')
  })

  it('names the day a goal aimed at, where the settings carry one', () => {
    const datePreset = settings({ goal: 'date', byDate: '2026-08-31' })

    expect(getStoppedWords(StopReason.PAST_DAY, datePreset, on)).toMatch(
      /^target date August.*passed$/,
    )
    // A preset counted without settings still reports what stopped it.
    expect(getStoppedWords(StopReason.PAST_DAY, null, on)).toBe('target date passed')
  })

  // The fifth of September in 2026 is a Saturday.
  it('names the day of the week carrying none of the load', () => {
    expect(getStoppedWords(StopReason.NO_LOAD, settings(), on)).toBe('no load on Saturday')
  })

  // A week at nothing names no day: there is no next one to name.
  it('names no day for a week carrying none of the load', () => {
    expect(getStoppedWords(StopReason.NO_WEEK, settings(), on)).toBe('no load on any day')
    expect(getStoppedWords(StopReason.NO_WEEK, null, on)).toBe('no load on any day')
  })
})

describe('what a goal evaluates to in words', () => {
  it('says how long a day runs', () => {
    expect(getGoalWords(settings({ minutesADay: 20 }), '2026-09-05')).toBe('20 minutes a day')
    expect(getGoalWords(settings({ minutesADay: 1 }), '2026-09-05')).toBe('1 minute a day')
    expect(getGoalWords(settings({ minutesADay: 0 }), '2026-09-05')).toBe('no time budget')
  })

  it('says the share requested of memory in hundredths', () => {
    expect(getGoalWords(settings({ goal: 'retention', retention: 0.9 }), '2026-09-05')).toBe(
      '90% remembered',
    )
    expect(getGoalWords(settings({ goal: 'retention', retention: 0.85 }), '2026-09-05')).toBe(
      '85% remembered',
    )
  })

  it('says the day, and how far off it is', () => {
    const transcriptText = getGoalWords(
      settings({ goal: 'date', byDate: '2026-09-30' }),
      '2026-09-12',
    )
    expect(transcriptText).toMatch(/^18 days to /)
  })

  it('says no day while the goal names none', () => {
    expect(getGoalWords(settings({ goal: 'date', byDate: '' }), '2026-09-12')).toBe(
      'no target date',
    )
  })

  it('says only the day once it is today or has passed', () => {
    const transcriptText = (byDate: string) =>
      getGoalWords(settings({ goal: 'date', byDate }), '2026-09-12')
    expect(transcriptText('2026-09-12')).toMatch(/^by \S/)
    expect(transcriptText('2026-09-10')).toMatch(/^by \S/)
    expect(transcriptText('2026-09-13')).toMatch(/^1 day to /)
  })
})

// A budget the goal does not name keeps its configured value and does not bind.
// Weighing the day against such a budget would measure against a limit that does
// not apply.
describe('how far through its day a preset remains', () => {
  // The user's defaults keep ten new cards a day while being steered by twenty
  // minutes, which is nearer sixty cards. Ten was never a wall.
  it('is weighed against the minutes alone where the minutes close the day', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 0, minutes: 20 }),
      closes: byMinutes,
      reviewed: 30,
      took: 5,
    })

    expect(getSpentShare(one)).toBeCloseTo(0.25)
    expect(isSpent(one)).toBe(false)
  })

  it('is weighed against the counts where the counts are what close it', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 45, minutes: 20 }),
      closes: byCounts,
      reviewed: 11,
      reviewedNew: 2,
      reviewedReviews: 9,
      took: 40,
    })

    expect(getSpentShare(one)).toBeCloseTo(0.2)
  })

  // Two budgets that close one day both limit it, and the day's progress is that of
  // the budget reached first. Answering every new card the day holds exhausts the
  // day on that side.
  it('remains at the fuller of two counts and not at their sum', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 200, minutes: 20 }),
      closes: byCounts,
      reviewed: 10,
      reviewedNew: 10,
      reviewedReviews: 0,
      took: 4,
    })

    expect(getSpentShare(one)).toBeCloseTo(1)
    expect(isSpent(one)).toBe(true)
  })

  // A goal of a date paces the new cards and leaves the reviews unbounded, so
  // the reviews close nothing and are weighed against nothing.
  it('leaves a day of reviews under a paced count inside its budget', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 0, minutes: 0 }),
      closes: { new: 'by_date', reviews: '', minutes: '' },
      reviewed: 50,
      reviewedNew: 10,
      reviewedReviews: 40,
      took: 30,
    })

    expect(getSpentShare(one)).toBeCloseTo(1)
  })

  it('keeps the fuller of them where both close the day', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 45, minutes: 20 }),
      closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: 'minutes_a_day' },
      reviewed: 11,
      took: 15,
    })

    expect(getSpentShare(one)).toBeCloseTo(0.75)
  })

  it('remains at nothing where no budget closes the day at all', () => {
    expect(getSpentShare(preset({ closes: CLOSES_NOTHING, reviewed: 30, took: 40 }))).toBe(0)
  })
})

describe('what starting a session on a preset would ask', () => {
  // The count is what a session will present, so it is printed as is, under every
  // goal.
  it('is the count itself, whatever budget the preset keeps', () => {
    expect(getLeftWords(preset({ cards: 34 }))).toBe('34 cards')
    expect(getLeftWords(preset({ cards: 34, budget: budget({ minutes: 0 }) }))).toBe('34 cards')
    expect(getLeftWords(preset({ cards: 1 }))).toBe('1 card')
  })

  // A tile with nothing to offer states the reason, as the deck rows below it do.
  it('says the day is full where the budget is what left it nothing', () => {
    expect(getLeftWords(preset({ cards: 0, reviewed: 55, took: 20 }))).toBe('daily limit reached')
  })

  it('says nothing fell due where the day retained none of its cards', () => {
    expect(getLeftWords(preset({ cards: 0, reviewed: 0, took: 0 }))).toBe('nothing due today')
  })
})

// The row of a deck and the letter drawn on it are one act, and both ask this.
describe('whether starting a session on a deck is offered', () => {
  const deck = (due: number, fresh = 0) => ({
    deck: 'decks/Words.md',
    reviewCards: 20,
    due,
    new: fresh,
    learned: 0,
    unbegun: 0,
  })
  const createPresets = (one?: Preset) => new Map(one ? [['decks/Words.md', one]] : [])

  it('is offered where the deck has cards due and its preset schedules something', () => {
    expect(canStart(deck(3), createPresets(preset()))).toBe(true)
    expect(canStart(deck(0, 1), createPresets(preset()))).toBe(true)
  })

  it('fails where the deck has no cards due', () => {
    expect(canStart(deck(0), createPresets(preset()))).toBe(false)
  })

  it('fails where the preset scheduling it schedules nothing', () => {
    expect(canStart(deck(3), createPresets(preset({ paused: 'no cards per day' })))).toBe(false)
  })

  it('is offered where nothing says which preset schedules the deck', () => {
    expect(canStart(deck(3), createPresets())).toBe(true)
  })
})

// The reasons stand in one column, under a tile and along a deck row, so they
// read in one voice.
describe('why a preset or a deck is asking nothing', () => {
  const on = '2026-09-05'

  it('is returned in one register: a clause in lower case, and never a sentence', () => {
    const response = [
      STOPPED.nothing,
      STOPPED.full,
      getStoppedWords(StopReason.NO_CARDS, null, on),
      getStoppedWords(StopReason.NO_MINUTES, null, on),
    ]

    for (const one of response) {
      expect(one).toBe(one.toLowerCase())
      expect(one).not.toMatch(/[.!?]$/)
    }
    // A day and a day of the week are names, and carry the only capital in one.
    expect(getStoppedWords(StopReason.PAST_DAY, settings({ byDate: '2026-09-30' }), on)).toMatch(
      / passed$/,
    )
    expect(getStoppedWords(StopReason.NO_LOAD, null, on)).toMatch(/^no load on \S+$/)
  })
})

describe('which preset schedules each deck', () => {
  it('gathers the decks of one preset and counts them together', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
        'decks/Roots.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault([
        { deck: 'decks/Words.md', due: 12, new: 4 },
        { deck: 'decks/Roots.md', due: 6, new: 2 },
      ]),
      '2026-09-05',
    )

    expect(one.presets.value).toHaveLength(1)
    expect(one.presets.value[0]).toMatchObject({
      path: 'Sanskrit.md',
      name: 'Sanskrit',
      decks: ['decks/Words.md', 'decks/Roots.md'],
      cards: 24,
      paused: '',
    })
    expect(one.byDeck.value.get('decks/Roots.md')?.name).toBe('Sanskrit')
  })

  // What the day holds under a preset, and what it has already come to, are the
  // count of the vault's own.
  it('takes today from the count of the vault, budget and all', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 40, new: 9 }],
        [
          presetDue({
            cards: 49,
            pending: 49,
            reviewed: 6,
            took: 2.5,
            new: 5,
            reviews: 23,
            minutes: 10,
          }),
        ],
      ),
      '2026-09-05',
    )

    // What the day leaves is the count's own figure, printed currently and
    // never worked out again from the budget.
    expect(one.presets.value[0]).toMatchObject({
      cards: 49,
      budget: { new: 5, reviews: 23, minutes: 10 },
      reviewed: 6,
      took: 2.5,
    })
  })

  // The tile prints what pressing it will ask, so the count it is given is the
  // count it shows.
  it('asks for exactly what is due in its decks, over all of them', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
        'decks/Roots.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault([
        { deck: 'decks/Words.md', due: 30, new: 7 },
        { deck: 'decks/Roots.md', due: 8, new: 2 },
      ]),
      '2026-09-05',
    )

    expect(one.presets.value[0]?.cards).toBe(47)
  })

  it('calls a deck naming no preset scheduled by the defaults', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: '', title: '', settings: settings() },
      }),
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.presets.value[0]).toMatchObject({ path: '', name: 'Defaults', cards: 4 })
  })

  it('holds nothing of the day for a preset that schedules nothing, and says why', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': {
          path: 'Stopped.md',
          title: 'Stopped',
          settings: settings({ goal: 'retention', newADay: 0, reviewsADay: 0 }),
          stopsOn: StopReason.NO_CARDS,
        },
      }),
    })

    // A preset that schedules nothing schedules no cards, so the count holds no
    // card of its decks and the row states the reason.
    await one.read(vault([{ deck: 'decks/Words.md', due: 0, new: 0 }]), '2026-09-05')

    expect(one.presets.value[0]).toMatchObject({
      paused: 'no cards per day',
      cards: 0,
    })
  })

  // A build that cannot work the presets of a vault shows none, and the decks
  // stand as they did.
  it('shows no preset where none could be read', async () => {
    const one = useVaultPresets({
      presets: {
        getVaultDeckPreset: () => Promise.reject(new Error('unimplemented')),
      },
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.presets.value).toHaveLength(0)
    expect(one.byDeck.value.size).toBe(0)
  })

  // No deck answers for a preset nothing points at, so the count is the only
  // place it can come from.
  it('gives a preset no deck points at a row of its own', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 3, new: 1 }],
        [
          presetDue({ cards: 4, pending: 4 }),
          presetDue({
            preset: 'Empty.md',
            title: 'Empty',
            decks: 0,
            cards: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
            closes: CLOSES_NOTHING,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value).toHaveLength(2)
    expect(one.presets.value[1]).toMatchObject({
      path: 'Empty.md',
      name: 'Empty',
      settings: null,
      decks: [],
      cards: 0,
      reviewed: 0,
      paused: '',
    })
    // Nothing points at it, so no deck is scheduled by it.
    expect(one.byDeck.value.get('decks/Words.md')?.name).toBe('Sanskrit')
    expect(one.byDeck.value.size).toBe(1)
  })

  // A preset no deck references has no settings, so the count's own verdict is the
  // only source of the reason it schedules nothing.
  it('says why a preset no deck points at schedules nothing', async () => {
    const one = useVaultPresets({ presets: createPresets({}) })

    await one.read(
      vault(
        [],
        [
          presetDue({ preset: 'Empty.md', title: 'Empty', decks: 0, cards: 0 }),
          presetDue({
            preset: 'Quiet.md',
            title: 'Quiet',
            decks: 0,
            cards: 0,
            stopsOn: StopReason.NO_MINUTES,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({ path: 'Empty.md', paused: '' })
    expect(one.presets.value[1]).toMatchObject({
      path: 'Quiet.md',
      paused: 'no time budget',
    })
  })

  // An empty deck has no cards due, so no deck references the preset it names and
  // the count is the only source of that row.
  it('gives a preset whose only deck is empty a row of its own', async () => {
    const one = useVaultPresets({ presets: createPresets({}) })

    await one.read(
      vault(
        [],
        [
          presetDue({
            preset: 'Empty.md',
            title: 'Empty',
            cards: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
            closes: CLOSES_NOTHING,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value).toHaveLength(1)
    expect(one.presets.value[0]).toMatchObject({ path: 'Empty.md', named: 1, reviewCards: 0 })
  })

  it('names a preset the count could not name after its file', async () => {
    const one = useVaultPresets({ presets: createPresets({}) })

    await one.read(
      vault(
        [],
        [
          presetDue({
            preset: 'goals/Empty.md',
            title: '',
            decks: 0,
            cards: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
            closes: CLOSES_NOTHING,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({ path: 'goals/Empty.md', name: 'Empty' })
  })

  // The count answered for the preset with its own figures, and a tile built
  // at nothing beside them is a day drawn as unbegun.
  it('draws a preset it could not read from the figures the count gave', async () => {
    const one = useVaultPresets({
      presets: { getVaultDeckPreset: async () => ({ error: ErrorCode.MISSING }) },
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 20, new: 2 }],
        [
          presetDue({
            cards: 40,
            pending: 22,
            reviewed: 6,
            took: 12,
            new: 0,
            reviews: 0,
            minutes: 20,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({
      cards: 22,
      reviewed: 6,
      took: 12,
      budget: { new: 0, reviews: 0, minutes: 20 },
      closes: byMinutes,
    })
  })

  it('says why a preset it could not read has no settings', async () => {
    const one = useVaultPresets({
      presets: { getVaultDeckPreset: async () => ({ error: ErrorCode.MISSING }) },
    })

    await one.read(
      vault([{ deck: 'decks/Words.md', due: 20, new: 2 }], [presetDue()]),
      '2026-09-05',
    )

    expect(one.presets.value[0]?.wrong).toBe('note not in the vault')
  })

  it('says nothing is wrong with a preset no deck of this vault answered for', async () => {
    const one = useVaultPresets({ presets: createPresets({}) })

    await one.read(vault([], [presetDue()]), '2026-09-05')

    expect(one.presets.value[0]?.wrong).toBe('')
  })

  // Errors in the file are resolved in the editor, and this window reports that
  // there is something to resolve.
  it('carries what was wrong in a preset it did read, once for all its decks', async () => {
    const one = useVaultPresets({
      presets: {
        async getVaultDeckPreset() {
          return {
            preset: {
              path: 'Sanskrit.md',
              title: 'Sanskrit',
              settings: createSettingsMessage(settings()),
              problems: ['`new_a_day` is not a number'],
              stopsOn: StopReason.NOTHING,
            },
          }
        },
      },
    })

    await one.read(
      vault([
        { deck: 'decks/Words.md', due: 3, new: 1 },
        { deck: 'decks/Roots.md', due: 2, new: 0 },
      ]),
      '2026-09-05',
    )

    expect(one.presets.value[0]?.wrong).toBe('`new_a_day` is not a number')
  })

  // A build whose count answers no preset leaves what the day holds to the
  // settings the preset itself was read with.
  it('falls back to the settings for what the day holds', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': {
          path: 'Sanskrit.md',
          title: 'Sanskrit',
          settings: settings({ newADay: 7, reviewsADay: 33, minutesADay: 12 }),
        },
      }),
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.presets.value[0]?.budget).toStrictEqual({ new: 7, reviews: 33, minutes: 12 })
  })

  // A user who navigates away while the presets of the previous vault are still
  // being requested receives nothing from that vault.
  it('holds no preset of a vault left while its presets were being requested', async () => {
    let answer: () => void = () => undefined
    const request = new Promise<void>((then) => {
      answer = then
    })
    const one = useVaultPresets({
      presets: {
        async getVaultDeckPreset() {
          await request
          return {
            preset: {
              path: 'Sanskrit.md',
              title: 'Sanskrit',
              settings: createSettingsMessage(settings()),
              problems: [],
              stopsOn: StopReason.NOTHING,
            },
          }
        },
      },
    })

    const readText = one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')
    one.forget()
    answer()
    await readText

    expect(one.presets.value).toHaveLength(0)
  })

  it('holds no preset for a vault a person has left', async () => {
    const one = useVaultPresets({ presets: createPresets({}) })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')
    one.forget()

    expect(one.presets.value).toHaveLength(0)
    expect(one.of.value).toBe('')
  })
})

describe('every question about a preset', () => {
  it('names the vault it is about', async () => {
    const named: string[] = []
    const answers = createPresets({
      'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
    })
    const one = useVaultPresets({
      presets: {
        ...answers,
        async getVaultDeckPreset(say) {
          named.push(say.vault)
          return answers.getVaultDeckPreset(say)
        },
      },
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(named).toEqual(['01A'])
  })
})

/** An application answering each deck named here with the error given, and every other deck with its preset. */
const createFailingPresets = (
  failing: Record<string, ErrorCode>,
  by: Parameters<typeof createPresets>[0] = {},
): PresetsClient => {
  const answers = createPresets(by)
  return {
    async getVaultDeckPreset(say) {
      const error = failing[say.deck]
      if (error !== undefined) return { error }
      return answers.getVaultDeckPreset(say)
    },
  }
}

describe('what a preset read from its decks carries of the count', () => {
  it('takes how many decks name it, its review cards and its answers from the count', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 3, new: 1 }],
        [
          presetDue({
            decks: 3,
            cards: 55,
            closes: byCounts,
            reviewed: 13,
            reviewedNew: 4,
            reviewedReviews: 9,
          }),
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({
      named: 3,
      reviewCards: 55,
      closes: byCounts,
      reviewedNew: 4,
      reviewedReviews: 9,
    })
  })

  it('relies on its own decks and on nothing answered where the count names it not', async () => {
    const one = useVaultPresets({
      presets: createPresets({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
        'decks/Roots.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault([
        { deck: 'decks/Words.md', due: 3, new: 1 },
        { deck: 'decks/Roots.md', due: 2, new: 0 },
      ]),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({
      named: 2,
      reviewCards: 0,
      closes: CLOSES_NOTHING,
      reviewedNew: 0,
      reviewedReviews: 0,
    })
  })

  it('says each thing wrong in the file once, the distinct ones apart', async () => {
    const response: Record<string, string> = {
      'decks/Words.md': '`new_a_day` is not a number',
      'decks/Roots.md': '`load` names no day',
    }
    const one = useVaultPresets({
      presets: {
        async getVaultDeckPreset({ deck }) {
          return {
            preset: {
              path: 'Sanskrit.md',
              title: 'Sanskrit',
              settings: createSettingsMessage(settings()),
              problems: [response[deck] ?? ''],
              stopsOn: StopReason.NOTHING,
            },
          }
        },
      },
    })

    await one.read(
      vault([
        { deck: 'decks/Words.md', due: 3, new: 1 },
        { deck: 'decks/Roots.md', due: 2, new: 0 },
      ]),
      '2026-09-05',
    )

    expect(one.presets.value[0]?.wrong).toBe('`new_a_day` is not a number; `load` names no day')
  })
})

describe('what is wrong with a preset no deck answered for', () => {
  it('is the error the decks that failed gave, whatever the decks that read gave', async () => {
    const one = useVaultPresets({
      presets: createFailingPresets(
        { 'decks/Roots.md': ErrorCode.MISSING },
        { 'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() } },
      ),
    })

    await one.read(
      vault(
        [
          { deck: 'decks/Words.md', due: 3, new: 1 },
          { deck: 'decks/Roots.md', due: 2, new: 0 },
        ],
        [presetDue(), presetDue({ preset: 'Latin.md', title: 'Latin' })],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[1]).toMatchObject({
      path: 'Latin.md',
      wrong: 'note not in the vault',
    })
  })

  it('is no one of the errors where the decks gave several', async () => {
    const one = useVaultPresets({
      presets: createFailingPresets({
        'decks/Words.md': ErrorCode.MISSING,
        'decks/Roots.md': ErrorCode.NOT_A_NOTE,
      }),
    })

    await one.read(
      vault(
        [
          { deck: 'decks/Words.md', due: 3, new: 1 },
          { deck: 'decks/Roots.md', due: 2, new: 0 },
        ],
        [presetDue({ decks: 2 })],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]?.wrong).toBe(PRESET_UNREADABLE_MESSAGE)
  })

  it('is nothing for a preset no deck names, whatever the other decks gave', async () => {
    const one = useVaultPresets({
      presets: createFailingPresets({ 'decks/Words.md': ErrorCode.MISSING }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 3, new: 1 }],
        [presetDue(), presetDue({ preset: 'Empty.md', title: 'Empty', decks: 0, cards: 0 })],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[1]).toMatchObject({ path: 'Empty.md', wrong: '' })
  })
})

describe('what is known of the presets of a vault', () => {
  /** An application that answers only once it is let go, with the same preset for every deck. */
  const createBlockedPresets = () => {
    let release: () => void = () => undefined
    let gate = Promise.resolve()
    const client: PresetsClient = {
      async getVaultDeckPreset() {
        await gate
        return {
          preset: {
            path: 'Sanskrit.md',
            title: 'Sanskrit',
            settings: createSettingsMessage(settings()),
            problems: [],
            stopsOn: StopReason.NOTHING,
          },
        }
      },
    }
    const hold = () => {
      gate = new Promise<void>((then) => {
        release = then
      })
    }
    return { client, hold, release: () => release() }
  }

  const other = (): VaultCardsDue => ({
    ...vault([{ deck: 'decks/Roots.md', due: 1, new: 0 }]),
    vault: '01B',
  })

  it('is known once a reading has run', async () => {
    const one = useVaultPresets({ presets: createBlockedPresets().client })

    expect(one.known.value).toBe(false)
    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.known.value).toBe(true)
  })

  it('stays known while the same vault is read again', async () => {
    const target = createBlockedPresets()
    const one = useVaultPresets({ presets: target.client })
    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    target.hold()
    const readText = one.read(vault([{ deck: 'decks/Words.md', due: 2, new: 1 }]), '2026-09-05')

    expect(one.known.value).toBe(true)
    target.release()
    await readText
    expect(one.known.value).toBe(true)
  })

  it('is unknown for another vault until its own reading has run', async () => {
    const target = createBlockedPresets()
    const one = useVaultPresets({ presets: target.client })
    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    target.hold()
    const readText = one.read(other(), '2026-09-05')

    expect(one.known.value).toBe(false)
    expect(one.of.value).toBe('01B')
    target.release()
    await readText
    expect(one.known.value).toBe(true)
  })

  it('holds nothing once it is read for no vault at all', async () => {
    const one = useVaultPresets({ presets: createBlockedPresets().client })
    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    await one.read(null, '2026-09-05')

    expect(one.presets.value).toHaveLength(0)
    expect(one.of.value).toBe('')
    expect(one.known.value).toBe(false)
  })
})
