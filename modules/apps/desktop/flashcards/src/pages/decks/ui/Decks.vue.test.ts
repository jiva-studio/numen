// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { StopReason } from '@numen/protocol'

import Decks from './Decks.vue'
import Presets from './Presets.vue'
import type { BudgetKeys, VaultCardsDue } from '@/entities/vault'
import type { Preset, Settings } from '../types'

const settings = (fields: Partial<Settings> = {}): Settings => ({
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  load: {},
  evenLoad: true,
  ...fields,
})

/** What closes the day of the fixture, which is steered by its minutes. */
const closes: BudgetKeys = { new: '', reviews: '', minutes: 'minutes_a_day' }

const preset = (fields: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  faces: 20,
  cards: 16,
  budget: { new: 10, reviews: 45, minutes: 20 },
  closes,
  answered: 0,
  answeredNew: 0,
  answeredReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...fields,
})

const vault: VaultCardsDue = {
  vault: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  counted: true,
  faces: 40,
  due: 12,
  new: 4,
  decks: [
    { deck: 'decks/Words.md', faces: 20, due: 8, new: 2, learned: 5, unbegun: 2 },
    { deck: 'decks/Roots.md', faces: 20, due: 4, new: 2, learned: 10, unbegun: 2 },
  ],
  presets: [
    {
      preset: 'Sanskrit.md',
      title: 'Sanskrit',
      decks: 1,
      cards: 20,
      owed: 16,
      answered: 0,
      answeredNew: 0,
      answeredReviews: 0,
      took: 0,
      new: 10,
      reviews: 45,
      minutes: 20,
      closes,
      stopsOn: StopReason.NOTHING,
    },
  ],
  unread: '',
  reading: false,
}

const mountDecks = (presets: readonly Preset[], over: VaultCardsDue = vault) =>
  mount(Decks, {
    props: {
      vault: over,
      days: new Map(),
      due: new Map(),
      presets,
      byDeck: new Map(presets.flatMap((one) => one.decks.map((deck) => [deck, one] as const))),
      scheduled: true,
      today: '2026-09-05',
    },
  })

describe('the decks of a vault', () => {
  it('says which preset schedules each deck', () => {
    const one = mountDecks([preset({ decks: ['decks/Words.md', 'decks/Roots.md'] })])

    expect(one.findAll('.decks__by').map((by) => by.text())).toEqual(['Sanskrit', 'Sanskrit'])
  })

  it('shows a deck of a preset that schedules nothing as not studied today', () => {
    const one = mountDecks([
      preset({ decks: ['decks/Words.md'], paused: 'no cards a day', cards: 0 }),
      preset({ path: 'Pali.md', name: 'Pali', decks: ['decks/Roots.md'] }),
    ])
    const rows = one.findAll('.decks__deck')

    expect(one.find('.decks__stopped').text()).toBe('no cards a day')
    expect(rows[0]?.attributes('disabled')).toBeDefined()
    expect(rows[1]?.attributes('disabled')).toBeUndefined()
  })

  it('leaves a deck alone where nothing says which preset schedules it', () => {
    const one = mountDecks([])

    expect(one.findAll('.decks__by')).toHaveLength(0)
    expect(one.findAll('.decks__stopped')).toHaveLength(0)
  })
})

describe('a deck with nothing waiting', () => {
  /** A vault whose one deck holds these cards and owes this much of them. */
  const createVault = (faces: number, due: number, fresh: number, unbegun = 0): VaultCardsDue => ({
    ...vault,
    due,
    new: fresh,
    decks: [{ deck: 'decks/Words.md', faces, due, new: fresh, learned: 0, unbegun }],
  })

  // Something was answered under the preset today and nothing of this deck is
  // left, which is the day's work met.
  it('says the day is done where the deck was answered and owes none', () => {
    const one = mountDecks([preset({ cards: 0, answered: 6, took: 3 })], createVault(20, 0, 0))

    expect(one.find('.decks__met').text()).toBe('Done today')
    expect(one.findAll('.due-count')).toHaveLength(1)
    expect(one.find('.decks__all').text()).toContain('0')
  })

  // Having nothing due is not having finished. Nothing was answered under this
  // preset today, so no deck of it has done anything.
  it('says a deck nothing fell due for has nothing, and never that it is done', () => {
    const one = mountDecks([preset({ cards: 0, answered: 0, took: 0 })], createVault(20, 0, 0))

    expect(one.find('.decks__stopped').text()).toBe('nothing today')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })

  // The budget was spent elsewhere under this preset, so this deck is asked
  // nothing. That is the preset's reason, the way a paused one's is.
  it('says the preset is full where its budget is what left the deck nothing', () => {
    const one = mountDecks([preset({ cards: 0, answered: 55, took: 20 })], createVault(20, 0, 0))

    expect(one.find('.decks__stopped').text()).toBe('the day is full')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })

  // Every card face here is one nobody has begun and the preset begins none a
  // day, so no next day picks any of them up. The preset is not stopped.
  it('says nothing here can be begun where the preset begins none a day', () => {
    const one = mountDecks(
      [preset({ cards: 0, budget: { new: 0, reviews: 200, minutes: 20 } })],
      createVault(20, 0, 0, 20),
    )

    expect(one.find('.decks__stopped').text()).toBe('no cards to begin')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })

  // A deck some of which has been begun has cards that come round, so its
  // quiet day is a quiet day.
  it('says nothing today where some of the deck has been begun', () => {
    const one = mountDecks(
      [preset({ cards: 0, budget: { new: 0, reviews: 200, minutes: 20 } })],
      createVault(20, 0, 0, 19),
    )

    expect(one.find('.decks__stopped').text()).toBe('nothing today')
  })

  // The preset begins cards a day, so the unbegun material is waiting on the
  // day and not on a setting.
  it('says nothing today where the preset does begin cards a day', () => {
    const one = mountDecks([preset({ cards: 0 })], createVault(20, 0, 0, 20))

    expect(one.find('.decks__stopped').text()).toBe('nothing today')
  })

  it('says the reason instead where the preset schedules nothing today', () => {
    const one = mountDecks([preset({ cards: 0, paused: 'no cards a day' })], createVault(20, 0, 0))

    expect(one.find('.decks__stopped').text()).toBe('no cards a day')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })

  it('says neither where the deck holds no card at all', () => {
    const one = mountDecks([preset({ cards: 0 })], createVault(0, 0, 0))

    expect(one.findAll('.decks__met')).toHaveLength(0)
    expect(one.findAll('.decks__stopped')).toHaveLength(0)
    // The pill on the button of the whole vault is the only count on the row.
    expect(one.find('.decks__deck').findAll('.due-count')).toHaveLength(0)
  })

  it('counts what is waiting where something is', () => {
    const one = mountDecks([preset()], createVault(20, 8, 2))

    expect(one.find('.decks__deck').find('.due-count').text()).toBe('10 to review')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })
})

// Starting a session on a preset is the deck screen's act, carried up from the tile
// to whatever opens a session.
describe('a preset pressed', () => {
  it('is passed on by the note it stands in', () => {
    const one = mountDecks([preset()])

    one.findComponent(Presets).vm.$emit('start', 'Sanskrit.md')

    expect(one.emitted('startPreset')).toStrictEqual([['Sanskrit.md']])
  })
})

// The tile over the list and the rows under it are drawn from the one reading
// of the day, so they cannot say different things about it.
describe('the tile and the decks under it', () => {
  it('agree: at nothing done, no deck of that preset claims to be done', () => {
    const over: VaultCardsDue = {
      ...vault,
      due: 10,
      new: 0,
      decks: [
        { deck: 'decks/Words.md', faces: 20, due: 10, new: 0, learned: 0, unbegun: 0 },
        { deck: 'decks/Roots.md', faces: 20, due: 0, new: 0, learned: 0, unbegun: 0 },
        { deck: 'decks/Stems.md', faces: 20, due: 0, new: 0, learned: 0, unbegun: 0 },
      ],
    }
    const one = mountDecks(
      [
        preset({
          decks: ['decks/Words.md', 'decks/Roots.md', 'decks/Stems.md'],
          named: 3,
          faces: 60,
          answered: 0,
          took: 0,
        }),
      ],
      over,
    )

    expect(one.find('.presets__done').text()).toBe('0%')
    expect(one.findAll('.decks__met')).toHaveLength(0)
    expect(one.findAll('.decks__stopped').map((said) => said.text())).toStrictEqual([
      'nothing today',
      'nothing today',
    ])
    expect(one.find('.decks__deck').find('.due-count').text()).toBe('10 to review')
  })
})
