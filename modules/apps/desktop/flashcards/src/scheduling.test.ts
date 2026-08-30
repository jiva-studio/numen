import { describe, expect, it } from 'vitest'
import { Goal } from '@numen/protocol'

import { goalWords, holds, load, named, paused, scheduling } from './scheduling'
import type { Asks, Budget, Preset, Settings } from './scheduling'
import type { Owing, PresetOwing } from './core'

const settings = (said: Partial<Settings> = {}): Settings => ({
  goal: Goal.MINUTES_A_DAY,
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  lightDays: [],
  evenLoad: true,
  ...said,
})

const budget = (said: Partial<Budget> = {}): Budget => ({
  new: 10,
  reviews: 200,
  minutes: 20,
  ...said,
})

const preset = (said: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  faces: 30,
  cards: 30,
  budget: budget(),
  answered: 0,
  took: 0,
  paused: '',
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
    const day = budget({ new: 10, reviews: 45 })
    expect(load(day, 200, 30)).toBe(55)
    expect(load(day, 12, 4)).toBe(16)
    expect(holds(day)).toBe(55)
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
          {
            preset: 'Sanskrit.md',
            title: 'Sanskrit',
            decks: 1,
            cards: 49,
            answered: 6,
            took: 2.5,
            new: 5,
            reviews: 23,
            minutes: 10,
          },
        ],
      ),
      '2026-09-05',
    )

    expect(one.presets.value[0]).toMatchObject({
      cards: 28,
      budget: { new: 5, reviews: 23, minutes: 10 },
      answered: 6,
      took: 2.5,
    })
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

    await one.read(vault([{ deck: 'decks/Words.md', due: 9, new: 3 }]), '2026-09-05')

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
          {
            preset: 'Sanskrit.md',
            title: 'Sanskrit',
            decks: 1,
            cards: 4,
            answered: 0,
            took: 0,
            new: 10,
            reviews: 200,
            minutes: 20,
          },
          {
            preset: 'Empty.md',
            title: 'Empty',
            decks: 0,
            cards: 0,
            answered: 0,
            took: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
          },
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
          {
            preset: 'Empty.md',
            title: 'Empty',
            decks: 1,
            cards: 0,
            answered: 0,
            took: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
          },
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
          {
            preset: 'goals/Empty.md',
            title: '',
            decks: 0,
            cards: 0,
            answered: 0,
            took: 0,
            new: 0,
            reviews: 0,
            minutes: 0,
          },
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
