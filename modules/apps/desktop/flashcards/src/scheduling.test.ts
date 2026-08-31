import { describe, expect, it } from 'vitest'
import { Goal } from '@numen/protocol'

import {
  CLOSES_NOTHING,
  goalWords,
  holds,
  leftWords,
  named,
  paused,
  scheduling,
  spent,
  through,
} from './scheduling'
import type { Asks, Budget, Closes, Preset, Settings } from './scheduling'
import type { Owing, PresetOwing } from './core'

const settings = (said: Partial<Settings> = {}): Settings => ({
  goal: Goal.MINUTES_A_DAY,
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  load: {},
  evenLoad: true,
  ...said,
})

const budget = (said: Partial<Budget> = {}): Budget => ({
  new: 10,
  reviews: 200,
  minutes: 20,
  ...said,
})

/** A preset steered by how long its day runs, which is the ordinary one. */
const byMinutes: Closes = { new: '', reviews: '', minutes: 'minutes_a_day' }

/** One steered by what it asks of memory, where the counts are what close it. */
const byCounts: Closes = { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' }

const preset = (said: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  faces: 30,
  cards: 30,
  budget: budget(),
  closes: byMinutes,
  answered: 0,
  took: 0,
  paused: '',
  ...said,
})

/** One preset of a vault as the count hands it over. */
const owing = (said: Partial<PresetOwing> = {}): PresetOwing => ({
  preset: 'Sanskrit.md',
  title: 'Sanskrit',
  decks: 1,
  cards: 30,
  owed: 0,
  answered: 0,
  took: 0,
  new: 10,
  reviews: 200,
  minutes: 20,
  closes: byMinutes,
  ...said,
})

const vault = (
  decks: readonly { deck: string; due: number; new: number }[],
  presets: readonly PresetOwing[] = [],
): Owing => ({
  vaultId: '01A',
  name: 'Vault',
  path: '/vaults/01A',
  faces: 0,
  due: 0,
  new: 0,
  decks: decks.map((one) => ({ deck: one.deck, faces: 0, due: one.due, new: one.new })),
  presets,
  unread: '',
})

/** An application answering one preset for each deck named here. */
const answering = (
  by: Record<string, { path: string; title: string; settings: Settings }>,
): Asks => ({
  async scheduling({ deck }) {
    const one = by[deck]
    if (!one) return {}
    return { preset: { path: one.path, title: one.title, settings: one.settings, problems: [] } }
  },
})

describe('what a day of a preset holds', () => {
  it('holds each kind of card up to what the day holds of it', () => {
    expect(holds(budget({ new: 10, reviews: 45 }))).toBe(55)
  })

  it('is nothing where the preset schedules nothing', () => {
    expect(paused(settings({ newADay: 0, reviewsADay: 0 }), '2026-09-05')).toBe('no cards a day')
    expect(paused(settings(), '2026-09-05')).toBe('')
  })

  it('is nothing past the day the goal names', () => {
    const by = settings({ goal: Goal.BY_DATE, byDate: '2026-08-31' })
    expect(paused(by, '2026-09-05')).toMatch(/has passed$/)
    expect(paused(by, '2026-08-31')).toBe('')
  })
})

describe('what a goal comes to in words', () => {
  it('says how long a day runs', () => {
    expect(goalWords(settings({ minutesADay: 20 }), '2026-09-05')).toBe('20 minutes a day')
    expect(goalWords(settings({ minutesADay: 0 }), '2026-09-05')).toBe('no budget in time')
  })

  it('says the share asked of memory', () => {
    expect(goalWords(settings({ goal: Goal.RETENTION, retention: 0.9 }), '2026-09-05')).toBe(
      'retention 0.90',
    )
  })

  it('says the day, and how far off it is', () => {
    const said = goalWords(
      settings({ goal: Goal.BY_DATE, byDate: '2026-09-30' }),
      '2026-09-12',
    )
    expect(said).toMatch(/^by /)
    expect(said).toMatch(/— 18 days$/)
  })
})

// A budget the goal does not name stands as the person left it and binds
// nothing. Weighing the day against one of those is measuring against a wall
// that is not there.
describe('how far through its day a preset stands', () => {
  // The user's defaults keep ten new cards a day while being steered by twenty
  // minutes, which is nearer sixty cards. Ten was never a wall.
  it('is weighed against the minutes alone where the minutes close the day', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 0, minutes: 20 }),
      closes: byMinutes,
      answered: 30,
      took: 5,
    })

    expect(through(one)).toBeCloseTo(0.25)
    expect(spent(one)).toBe(false)
  })

  it('is weighed against the counts where the counts are what close it', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 45, minutes: 20 }),
      closes: byCounts,
      answered: 11,
      took: 40,
    })

    expect(through(one)).toBeCloseTo(0.2)
  })

  it('keeps the fuller of them where both close the day', () => {
    const one = preset({
      budget: budget({ new: 10, reviews: 45, minutes: 20 }),
      closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: 'minutes_a_day' },
      answered: 11,
      took: 15,
    })

    expect(through(one)).toBeCloseTo(0.75)
  })

  it('stands at nothing where no budget closes the day at all', () => {
    expect(through(preset({ closes: CLOSES_NOTHING, answered: 30, took: 40 }))).toBe(0)
  })
})

describe('what sitting down to a preset would ask', () => {
  // The count is what the sitting will put in front of a person, so it is
  // printed as it stands, under every goal.
  it('is the count itself, whatever budget the preset keeps', () => {
    expect(leftWords(preset({ cards: 34 }))).toBe('34 cards')
    expect(leftWords(preset({ cards: 34, budget: budget({ minutes: 0 }) }))).toBe('34 cards')
    expect(leftWords(preset({ cards: 1 }))).toBe('1 card')
  })

  it('is nothing at all where it would ask nothing', () => {
    expect(leftWords(preset({ cards: 0 }))).toBe('')
  })
})

describe('which preset schedules each deck', () => {
  it('gathers the decks of one preset and counts them together', async () => {
    const one = scheduling({
      presets: answering({
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
    const one = scheduling({
      presets: answering({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 40, new: 9 }],
        [
          owing({
            cards: 49,
            owed: 49,
            answered: 6,
            took: 2.5,
            new: 5,
            reviews: 23,
            minutes: 10,
          }),
        ],
      ),
      '2026-09-05',
    )

    // What the day leaves is the count's own figure, printed as it stands and
    // never worked out again from the budget.
    expect(one.presets.value[0]).toMatchObject({
      cards: 49,
      budget: { new: 5, reviews: 23, minutes: 10 },
      answered: 6,
      took: 2.5,
    })
  })

  // The tile prints what pressing it will ask, so the count it is given is the
  // count it shows.
  it('asks for exactly what its decks owe, over all of them', async () => {
    const one = scheduling({
      presets: answering({
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
    const one = scheduling({
      presets: answering({
        'decks/Words.md': { path: '', title: '', settings: settings() },
      }),
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.presets.value[0]).toMatchObject({ path: '', name: 'The defaults', cards: 4 })
  })

  it('holds nothing of the day for a preset that schedules nothing, and says why', async () => {
    const one = scheduling({
      presets: answering({
        'decks/Words.md': {
          path: 'Stopped.md',
          title: 'Stopped',
          settings: settings({ newADay: 0, reviewsADay: 0 }),
        },
      }),
    })

    // A preset that schedules nothing is asked for nothing, so the count owes
    // no card of its decks and the row says why instead.
    await one.read(vault([{ deck: 'decks/Words.md', due: 0, new: 0 }]), '2026-09-05')

    expect(one.presets.value[0]).toMatchObject({
      paused: 'no cards a day',
      cards: 0,
    })
  })

  // A build that cannot work the presets of a vault shows none, and the decks
  // stand as they did.
  it('shows no preset where none could be read', async () => {
    const one = scheduling({
      presets: {
        scheduling: () => Promise.reject(new Error('unimplemented')),
      },
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(one.presets.value).toHaveLength(0)
    expect(one.byDeck.value.size).toBe(0)
  })

  // No deck answers for a preset nothing points at, so the count is the only
  // place it can come from.
  it('gives a preset no deck points at a row of its own', async () => {
    const one = scheduling({
      presets: answering({
        'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
      }),
    })

    await one.read(
      vault(
        [{ deck: 'decks/Words.md', due: 3, new: 1 }],
        [
          owing({ cards: 4, owed: 4 }),
          owing({
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
      answered: 0,
      paused: '',
    })
    // Nothing points at it, so no deck is scheduled by it.
    expect(one.byDeck.value.get('decks/Words.md')?.name).toBe('Sanskrit')
    expect(one.byDeck.value.size).toBe(1)
  })

  // An empty deck owes nothing, so no deck answers for the preset it names and
  // the count is the only place that row can come from.
  it('gives a preset whose only deck is empty a row of its own', async () => {
    const one = scheduling({ presets: answering({}) })

    await one.read(
      vault(
        [],
        [
          owing({
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
    expect(one.presets.value[0]).toMatchObject({ path: 'Empty.md', named: 1, faces: 0 })
  })

  it('names a preset the count could not name after its file', async () => {
    const one = scheduling({ presets: answering({}) })

    await one.read(
      vault(
        [],
        [
          owing({
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

  it('holds no preset for a vault a person has left', async () => {
    const one = scheduling({ presets: answering({}) })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')
    one.forget()

    expect(one.presets.value).toHaveLength(0)
    expect(one.of.value).toBe('')
  })

  it('shows a day as the application writes one', () => {
    expect(named(new Date(2026, 8, 5))).toBe('2026-09-05')
  })
})

describe('every question about a preset', () => {
  it('names the vault it is about', async () => {
    const named: string[] = []
    const answers = answering({
      'decks/Words.md': { path: 'Sanskrit.md', title: 'Sanskrit', settings: settings() },
    })
    const one = scheduling({
      presets: {
        ...answers,
        async scheduling(say) {
          named.push(say.vaultId)
          return answers.scheduling(say)
        },
      },
    })

    await one.read(vault([{ deck: 'decks/Words.md', due: 3, new: 1 }]), '2026-09-05')

    expect(named).toEqual(['01A'])
  })
})
