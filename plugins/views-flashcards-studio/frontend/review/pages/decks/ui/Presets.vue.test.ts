// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Presets from './Presets.vue'
import type { BudgetKeys } from '../../../entities/vault'
import type { Preset, Settings } from '../types'

const settings = (fields: Partial<Settings> = {}): Settings => ({
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  load: {},
  hasEvenLoad: true,
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
  reviewCards: 40,
  cards: 22,
  budget: { new: 10, reviews: 45, minutes: 20 },
  closes,
  reviewed: 11,
  reviewedNew: 2,
  reviewedReviews: 9,
  took: 4,
  paused: '',
  wrong: '',
  ...fields,
})

const mountPresets = (presets: readonly Preset[], today = '2026-09-05') =>
  mount(Presets, { props: { presets, today } })

describe('what the goals come to today', () => {
  it('says a preset, its goal, and how far through the day it remains', () => {
    const one = mountPresets([preset()])

    expect(one.text()).toContain('Sanskrit')
    expect(one.text()).toContain('20 minutes a day')
    // Eleven cards of fifty-five is a fifth of the day, and four minutes of
    // twenty is the same fifth.
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  // A text figure states the progress: a progress bar would take a row of the
  // screen to repeat what the number already states.
  it('says how far through the day it is in words alone, and draws nothing', () => {
    const one = mountPresets([preset()])

    // A meter has no text content, so none is rendered: every element of the tile
    // contains text.
    const tile = one.get('.presets__preset')
    expect(tile.findAll('*').filter((each) => each.text() === '')).toStrictEqual([])
    expect(tile.text()).toBe('Sanskrit20 minutes a day20%22 cards')
  })

  // The tile reads across: the name with the goal under it, and what the day
  // evaluates to at the far end of the line.
  it('remains the name and goal together, and the figure apart from them', () => {
    const tile = mountPresets([preset()]).find('.presets__preset')
    const response = tile.find('.presets__transcript')

    expect(response.find('.presets__name').text()).toBe('Sanskrit')
    expect(response.find('.presets__goal').text()).toBe('20 minutes a day')
    // The figure is the tile's own child, beside that pair and not under them.
    expect(response.find('.presets__done').exists()).toBe(false)
    expect(tile.find('.presets__done').text()).toBe('20%')
  })

  // Each preset is an island of its own, the way a deck in the list below is.
  it('gives every preset a tile of its own', () => {
    const one = mountPresets([
      preset({ path: 'A.md', name: 'Sanskrit' }),
      preset({ path: 'B.md', name: 'Pali' }),
      preset({ path: 'C.md', name: 'Greek' }),
    ])
    const presetElements = one.findAll('.presets__preset')

    expect(presetElements).toHaveLength(3)
    expect(presetElements.map((tile) => tile.find('.presets__name').text())).toStrictEqual([
      'Sanskrit',
      'Pali',
      'Greek',
    ])
    // The heading, the goal below it, and the progress figure.
    expect(presetElements[1]?.find('.presets__goal').text()).toBe('20 minutes a day')
    expect(presetElements[1]?.find('.presets__done').text()).toBe('20%')
  })

  // Progress equals the fraction of the budget that closes the day.
  it('remains as far through as the minutes it is steered by', () => {
    const one = mountPresets([preset({ reviewed: 11, took: 15 })])

    expect(one.find('.presets__done').text()).toBe('75%')
  })

  it('relies on the cards alone where the preset keeps no budget in time', () => {
    const one = mountPresets([
      preset({
        settings: settings({ goal: 'retention', minutesADay: 0 }),
        budget: { new: 10, reviews: 45, minutes: 0 },
        closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' },
        reviewed: 11,
        took: 40,
      }),
    ])

    expect(one.text()).toContain('90% remembered')
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  it('is all of the day where a budget has been met exactly', () => {
    const one = mountPresets([preset({ reviewed: 55, took: 20 })])

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
        reviewed: 66,
        reviewedNew: 6,
        reviewedReviews: 60,
      }),
    ])
    const response = one.find('.presets__done')

    expect(response.text()).toBe('over budget')
    expect(response.attributes('data-over')).toBe('')
  })

  it('says the same of a day that has run past its minutes', () => {
    const one = mountPresets([preset({ reviewed: 0, took: 31 })])

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
        reviewed: 30,
        took: 5,
      }),
    ])
    const response = one.find('.presets__done')

    expect(response.text()).toBe('25%')
    expect(response.attributes('data-over')).toBeUndefined()
  })

  // The reverse case: a preset steered by a retention goal ignores its minutes
  // budget.
  it('weighs it against the counts where the counts are what close the day', () => {
    const one = mountPresets([
      preset({
        settings: settings({ goal: 'retention' }),
        budget: { new: 10, reviews: 45, minutes: 5 },
        closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' },
        reviewed: 11,
        took: 40,
      }),
    ])

    expect(one.find('.presets__done').text()).toBe('20%')
  })

  // The tile shows the goal and the progress of the day only.
  it('says nothing of counts, of what closes the day, or of the week', () => {
    const response = mountPresets([
      preset({ path: 'A.md', settings: settings({ load: { sat: 50 } }) }),
      preset({ path: 'B.md', name: 'Pali' }),
    ]).text()

    expect(response).not.toContain('of 55')
    expect(response).not.toContain('close the day')
    expect(response).not.toContain('Today:')
    expect(response).not.toContain('light in')
  })

  // The tile shows why the preset schedules nothing in place of its goal, so the
  // reason takes the place of the goal.
  it('greys a preset that schedules nothing and says why in place of the goal', () => {
    const one = mountPresets([preset({ paused: 'no cards per day' })])

    expect(one.find('.presets__preset--paused').exists()).toBe(true)
    expect(one.find('.presets__goal').text()).toBe('no cards per day')
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
        reviewed: 6,
        took: 4,
        wrong: 'note not in the vault',
      }),
    ])

    expect(one.find('.presets__done').text()).toBe('20%')
    expect(one.find('.presets__left').text()).toBe('22 cards')
    expect(one.find('.presets__preset').attributes('disabled')).toBeUndefined()
  })

  it('says what is wrong with a preset where its goal would stand', () => {
    const one = mountPresets([preset({ settings: null, wrong: 'note not in the vault' })])

    expect(one.find('.presets__wrong').text()).toBe('note not in the vault')
  })

  it('says nothing of what is wrong where nothing is', () => {
    expect(mountPresets([preset()]).findAll('.presets__wrong')).toHaveLength(0)
  })

  // This screen shows the day's progress. A preset that no deck points at is
  // still being set up, and the editor's preset tab is where it is shown.
  it('leaves out a preset no deck points at', () => {
    const one = mountPresets([
      preset({
        decks: [],
        named: 0,
        reviewCards: 0,
        settings: null,
        cards: 0,
        reviewed: 0,
        took: 0,
      }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('Nothing points here')
  })

  it('leaves out a preset whose decks hold no card between them', () => {
    const one = mountPresets([
      preset({
        decks: [],
        named: 2,
        reviewCards: 0,
        settings: null,
        cards: 0,
        reviewed: 0,
        took: 0,
      }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('no cards')
  })

  // A preset that schedules nothing today is listed, because its decks are
  // listed with that status and the reason must be visible.
  it('keeps a preset that schedules nothing today, alongside the ones that do', () => {
    const one = mountPresets([
      preset({ path: 'A.md', name: 'Sanskrit' }),
      preset({ path: 'B.md', name: 'Stopped', paused: 'no cards per day' }),
      preset({ path: 'C.md', named: 0, reviewCards: 0, settings: null }),
    ])
    const presetElements = one.findAll('.presets__preset')

    expect(presetElements).toHaveLength(2)
    expect(presetElements[1]?.text()).toBe('Stoppedno cards per day')
    expect(presetElements[1]?.find('.presets__done').exists()).toBe(false)
  })

  // The decks naming no preset are a tile like any other, and it carries no act.
  it('draws the tile of the defaults as an ordinary preset', () => {
    const one = mountPresets([preset({ path: '', name: 'Defaults' })])

    expect(one.text()).toContain('Defaults')
    expect(one.findAll('.presets__done')).toHaveLength(1)
    // Starting a session on it is the whole tile, the way it is for a preset that
    // names itself: the tile carries no act of its own beside that.
    expect(one.get('.presets__preset').findAll('button, a, [role="button"]')).toHaveLength(0)
  })

  it('remains aside where the vault has no preset to show', () => {
    expect(mountPresets([]).find('.presets').exists()).toBe(false)
  })
})

// The tiles form a list with an accessible name, and the figures are a live
// region that updates after a session ends.
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

// The figure states the progress of the day; the count states the cards
// remaining, which is the number that informs starting a session.
describe('what pressing a preset would ask', () => {
  // The tile shows the number of cards a session started from it would present,
  // so the number is exact under every goal.
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

  // A tile that cannot be pressed states the reason, like the deck rows below it.
  it('says why in its place where there is nothing to ask', () => {
    expect(
      mountPresets([preset({ cards: 0, reviewed: 55, took: 20 })])
        .find('.presets__left')
        .text(),
    ).toBe('daily limit reached')
    expect(
      mountPresets([preset({ cards: 0, reviewed: 0, took: 0 })])
        .find('.presets__left')
        .text(),
    ).toBe('nothing due today')
  })

  it('says nothing of a count where the preset schedules nothing today', () => {
    const one = mountPresets([preset({ cards: 0, paused: 'no cards per day' })])

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

  // Decks that name no preset use the defaults like any other preset, and the
  // defaults have no note path.
  it('names the defaults by the nothing they stand in', async () => {
    const one = mountPresets([preset({ path: '', name: 'Defaults' })])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toStrictEqual([['']])
  })

  // A preset with no cards available shows the reason above and is not pressable.
  it('offers no session where the day under it is already done', async () => {
    const one = mountPresets([preset({ cards: 0 })])

    expect(getPressable(one)).toStrictEqual([false])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toBeUndefined()
  })

  it('offers no session where the preset schedules nothing today', () => {
    expect(
      getPressable(mountPresets([preset({ cards: 0, paused: 'no cards per day' })])),
    ).toStrictEqual([false])
  })
})
