// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Presets from './Presets.vue'
import type { BudgetKeys } from '@/entities/vault'
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
  faces: 40,
  cards: 22,
  budget: { new: 10, reviews: 45, minutes: 20 },
  closes,
  answered: 11,
  answeredNew: 2,
  answeredReviews: 9,
  took: 4,
  paused: '',
  wrong: '',
  ...fields,
})

const mountPresets = (presets: readonly Preset[], today = '2026-09-05') =>
  mount(Presets, { props: { presets, today } })

describe('what the goals come to today', () => {
  it('says a preset, its goal, and how far through the day it stands', () => {
    const one = mountPresets([preset()])

    expect(one.text()).toContain('Sanskrit')
    expect(one.text()).toContain('20 minutes a day')
    // Eleven cards of fifty-five is a fifth of the day, and four minutes of
    // twenty is the same fifth.
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  // The screen is scanned, and a bar reading a fifth costs a row of it to say
  // what the number already says.
  it('says how far through the day it is in words alone, and draws nothing', () => {
    const one = mountPresets([preset()])

    // A meter says nothing: it is drawn to be looked at. Everything the tile
    // draws carries words, so there is none.
    const tile = one.get('.presets__preset')
    expect(tile.findAll('*').filter((each) => each.text() === '')).toStrictEqual([])
    expect(tile.text()).toBe('Sanskrit20 minutes a day20%22 cards')
  })

  // The tile reads across: the name with the goal under it, and what the day
  // comes to at the far end of the line.
  it('stands the name and goal together, and the figure apart from them', () => {
    const tile = mountPresets([preset()]).find('.presets__preset')
    const said = tile.find('.presets__said')

    expect(said.find('.presets__name').text()).toBe('Sanskrit')
    expect(said.find('.presets__goal').text()).toBe('20 minutes a day')
    // The figure is the tile's own child, beside that pair and not under them.
    expect(said.find('.presets__done').exists()).toBe(false)
    expect(tile.find('.presets__done').text()).toBe('20%')
  })

  // Each preset is an island of its own, the way a deck in the list below is.
  it('gives every preset a tile of its own', () => {
    const one = mountPresets([
      preset({ path: 'A.md', name: 'Sanskrit' }),
      preset({ path: 'B.md', name: 'Pali' }),
      preset({ path: 'C.md', name: 'Greek' }),
    ])
    const drawn = one.findAll('.presets__preset')

    expect(drawn).toHaveLength(3)
    expect(drawn.map((tile) => tile.find('.presets__name').text())).toStrictEqual([
      'Sanskrit',
      'Pali',
      'Greek',
    ])
    // The heading, the goal under it, and the number a person looks for.
    expect(drawn[1]?.find('.presets__goal').text()).toBe('20 minutes a day')
    expect(drawn[1]?.find('.presets__done').text()).toBe('20%')
  })

  // A person is as far through their day as the budget closing it says.
  it('stands as far through as the minutes it is steered by', () => {
    const one = mountPresets([preset({ answered: 11, took: 15 })])

    expect(one.find('.presets__done').text()).toBe('75%')
  })

  it('stands on the cards alone where the preset keeps no budget in time', () => {
    const one = mountPresets([
      preset({
        settings: settings({ goal: 'retention', minutesADay: 0 }),
        budget: { new: 10, reviews: 45, minutes: 0 },
        closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' },
        answered: 11,
        took: 40,
      }),
    ])

    expect(one.text()).toContain('90% remembered')
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  it('is all of the day where a budget has been met exactly', () => {
    const one = mountPresets([preset({ answered: 55, took: 20 })])

    expect(one.find('.presets__done').text()).toBe('100%')
  })

  // Sixty-six answers against thirteen cards is not a day that is done, and a
  // round number would read as one.
  it('says a day answered past its budget is over it, and never a percentage', () => {
    const one = mountPresets([
      preset({
        settings: settings({ goal: 'retention' }),
        budget: { new: 3, reviews: 10, minutes: 20 },
        closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' },
        answered: 66,
        answeredNew: 6,
        answeredReviews: 60,
      }),
    ])
    const said = one.find('.presets__done')

    expect(said.text()).toBe('over budget')
    expect(said.attributes('data-over')).toBe('')
  })

  it('says the same of a day that has run past its minutes', () => {
    const one = mountPresets([preset({ answered: 0, took: 31 })])

    expect(one.find('.presets__done').text()).toBe('over budget')
  })

  // The user's defaults keep ten new cards a day while being steered by twenty
  // minutes, which is nearer sixty cards. Weighed against the ten that bind
  // nothing, thirty answers reads as a day run over; weighed against the
  // minutes that do close it, the day is a quarter done.
  it('weighs the day against the budget that closes it, never one left inert', () => {
    const one = mountPresets([
      preset({
        budget: { new: 10, reviews: 0, minutes: 20 },
        closes: { new: '', reviews: '', minutes: 'minutes_a_day' },
        answered: 30,
        took: 5,
      }),
    ])
    const said = one.find('.presets__done')

    expect(said.text()).toBe('25%')
    expect(said.attributes('data-over')).toBeUndefined()
  })

  // The same the other way about: a preset steered by what it asks of memory
  // keeps its minutes as the person left them.
  it('weighs it against the counts where the counts are what close the day', () => {
    const one = mountPresets([
      preset({
        settings: settings({ goal: 'retention' }),
        budget: { new: 10, reviews: 45, minutes: 5 },
        closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' },
        answered: 11,
        took: 40,
      }),
    ])

    expect(one.find('.presets__done').text()).toBe('20%')
  })

  // The block says the goal and how far through it the day is, and nothing
  // else.
  it('says nothing of counts, of what closes the day, or of the week', () => {
    const said = mountPresets([
      preset({ path: 'A.md', settings: settings({ load: { sat: 50 } }) }),
      preset({ path: 'B.md', name: 'Pali' }),
    ]).text()

    expect(said).not.toContain('of 55')
    expect(said).not.toContain('close the day')
    expect(said).not.toContain('Today:')
    expect(said).not.toContain('light in')
  })

  // What a person needs from the tile is why it schedules nothing, not what it
  // was aiming at, so the reason stands where the goal would.
  it('greys a preset that schedules nothing and says why in place of the goal', () => {
    const one = mountPresets([preset({ paused: 'no cards a day' })])

    expect(one.find('.presets__preset--paused').exists()).toBe(true)
    expect(one.find('.presets__goal').text()).toBe('no cards a day')
    expect(one.text()).not.toContain('20 minutes a day')
    expect(one.findAll('.presets__done')).toHaveLength(0)
    expect(one.text()).not.toContain('%')
  })

  // The settings could not be read, and the count answered for the preset all
  // the same. The day is drawn from what it gave.
  it('draws a preset whose settings could not be read from what the count gave', () => {
    const one = mountPresets([
      preset({
        settings: null,
        cards: 22,
        answered: 6,
        took: 4,
        wrong: 'that note is not in the vault',
      }),
    ])

    expect(one.find('.presets__done').text()).toBe('20%')
    expect(one.find('.presets__left').text()).toBe('22 cards')
    expect(one.find('.presets__preset').attributes('disabled')).toBeUndefined()
  })

  it('says what is wrong with a preset where its goal would stand', () => {
    const one = mountPresets([preset({ settings: null, wrong: 'that note is not in the vault' })])

    expect(one.find('.presets__wrong').text()).toBe('that note is not in the vault')
  })

  it('says nothing of what is wrong where nothing is', () => {
    expect(mountPresets([preset()]).findAll('.presets__wrong')).toHaveLength(0)
  })

  // This screen is what a person's day comes to. A preset nothing points at is
  // a file being set up, and the editor's preset tab is where it is read.
  it('leaves out a preset no deck points at', () => {
    const one = mountPresets([
      preset({ decks: [], named: 0, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('Nothing points here')
  })

  it('leaves out a preset whose decks hold no card between them', () => {
    const one = mountPresets([
      preset({ decks: [], named: 2, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('no cards')
  })

  // A preset that schedules nothing today is another matter: its decks are in
  // the list saying so, and a person needs to see why.
  it('keeps a preset that schedules nothing today, beside the ones that do', () => {
    const one = mountPresets([
      preset({ path: 'A.md', name: 'Sanskrit' }),
      preset({ path: 'B.md', name: 'Stopped', paused: 'no cards a day' }),
      preset({ path: 'C.md', named: 0, faces: 0, settings: null }),
    ])
    const drawn = one.findAll('.presets__preset')

    expect(drawn).toHaveLength(2)
    expect(drawn[1]?.text()).toBe('Stoppedno cards a day')
    expect(drawn[1]?.find('.presets__done').exists()).toBe(false)
  })

  // The decks naming no preset are a tile like any other, and it carries no act.
  it('draws the tile of the defaults as an ordinary preset', () => {
    const one = mountPresets([preset({ path: '', name: 'The defaults' })])

    expect(one.text()).toContain('The defaults')
    expect(one.findAll('.presets__done')).toHaveLength(1)
    // Starting a session on it is the whole tile, the way it is for a preset that
    // names itself: the tile carries no act of its own beside that.
    expect(one.get('.presets__preset').findAll('button, a, [role="button"]')).toHaveLength(0)
  })

  it('stands aside where the vault has no preset to show', () => {
    expect(mountPresets([]).find('.presets').exists()).toBe(false)
  })
})

// The tiles are a list with nothing over them, and the figures on them change
// under a person who has just left a session.
describe('the tiles read aloud', () => {
  it('names the list', () => {
    expect(mountPresets([preset()]).find('.presets__list').attributes('aria-label')).toBe(
      'What the goals of this vault come to today',
    )
  })

  it('says the figures again as they change', () => {
    expect(mountPresets([preset()]).find('.presets__figures').attributes('aria-live')).toBe(
      'polite',
    )
  })
})

// The figure says how far through the day a person is; the count says what is
// left to do, which is what they are deciding on.
describe('what pressing a preset would ask', () => {
  // The tile prints what pressing it will put in front of a person, so the
  // number is exact and carries no hedge under any goal.
  it('counts it under the figure, exactly and without a hedge', () => {
    const one = mountPresets([preset({ cards: 22 })])

    expect(one.find('.presets__left').text()).toBe('22 cards')
  })

  it('counts it the same where the preset keeps no budget in time', () => {
    const one = mountPresets([
      preset({ cards: 22, budget: { new: 10, reviews: 45, minutes: 0 }, took: 0 }),
    ])

    expect(one.find('.presets__left').text()).toBe('22 cards')
  })

  it('says one card as one', () => {
    expect(
      mountPresets([preset({ cards: 1 })])
        .find('.presets__left')
        .text(),
    ).toBe('1 card')
  })

  // A tile that cannot be pressed says why, as the deck rows under it do.
  it('says why in its place where there is nothing to ask', () => {
    expect(
      mountPresets([preset({ cards: 0, answered: 55, took: 20 })])
        .find('.presets__left')
        .text(),
    ).toBe('the day is full')
    expect(
      mountPresets([preset({ cards: 0, answered: 0, took: 0 })])
        .find('.presets__left')
        .text(),
    ).toBe('nothing today')
  })

  it('says nothing of a count where the preset schedules nothing today', () => {
    const one = mountPresets([preset({ cards: 0, paused: 'no cards a day' })])

    expect(one.findAll('.presets__left')).toHaveLength(0)
  })
})

describe('starting a session on a preset', () => {
  /** Which tiles can be pressed, in the order they stand. */
  const getPressable = (one: ReturnType<typeof mountPresets>): readonly boolean[] =>
    one.findAll('.presets__preset').map((tile) => tile.attributes('disabled') === undefined)

  it('is offered by a preset with cards to ask, and names the preset pressed', async () => {
    const one = mountPresets([
      preset({ path: 'Sanskrit.md' }),
      preset({ path: 'Pali.md', name: 'Pali' }),
    ])
    const tiles = one.findAll('.presets__preset')

    expect(getPressable(one)).toStrictEqual([true, true])

    await tiles[1]?.trigger('click')

    expect(one.emitted('start')).toStrictEqual([['Pali.md']])
  })

  // The decks naming no preset are sat down to the way the rest are, and the
  // note they stand under is no note at all.
  it('names the defaults by the nothing they stand in', async () => {
    const one = mountPresets([preset({ path: '', name: 'The defaults' })])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toStrictEqual([['']])
  })

  // A preset with nothing to offer has said above why, and is not pressed.
  it('offers no session where the day under it is already done', async () => {
    const one = mountPresets([preset({ cards: 0 })])

    expect(getPressable(one)).toStrictEqual([false])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toBeUndefined()
  })

  it('offers no session where the preset schedules nothing today', () => {
    expect(getPressable(mountPresets([preset({ cards: 0, paused: 'no cards a day' })]))).toStrictEqual(
      [false],
    )
  })
})
