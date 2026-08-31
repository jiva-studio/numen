// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Goal } from '@numen/protocol'

import Presets from './Presets.vue'
import type { Preset, Settings } from './scheduling'

const settings = (said: Partial<Settings> = {}): Settings => ({
  goal: Goal.MINUTES_A_DAY,
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  lightDays: [],
  evenLoad: true,
  ...said,
})

const preset = (said: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  faces: 40,
  cards: 22,
  budget: { new: 10, reviews: 45, minutes: 20 },
  answered: 11,
  took: 4,
  paused: '',
  ...said,
})

const shown = (presets: readonly Preset[], today = '2026-09-05') =>
  mount(Presets, { props: { presets, today } })

describe('what the goals come to today', () => {
  it('says a preset, its goal, and how far through the day it stands', () => {
    const one = shown([preset()])

    expect(one.text()).toContain('Sanskrit')
    expect(one.text()).toContain('20 minutes a day')
    // Eleven cards of fifty-five is a fifth of the day, and four minutes of
    // twenty is the same fifth.
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  // The screen is scanned, and a bar reading a fifth costs a row of it to say
  // what the number already says.
  it('says how far through the day it is in words alone, and draws nothing', () => {
    const one = shown([preset()])

    expect(one.findAll('.presets__track')).toHaveLength(0)
    expect(one.findAll('.presets__through')).toHaveLength(0)
    expect(one.find('.presets__preset').text()).toBe('Sanskrit20 minutes a day20%about 22 cards')
  })

  // The tile reads across: the name with the goal under it, and what the day
  // comes to at the far end of the line.
  it('stands the name and goal together, and the figure apart from them', () => {
    const tile = shown([preset()]).find('.presets__preset')
    const said = tile.find('.presets__said')

    expect(said.find('.presets__name').text()).toBe('Sanskrit')
    expect(said.find('.presets__goal').text()).toBe('20 minutes a day')
    // The figure is the tile's own child, beside that pair and not under them.
    expect(said.find('.presets__done').exists()).toBe(false)
    expect(tile.find('.presets__done').text()).toBe('20%')
  })

  // Each preset is an island of its own, the way a deck in the list below is.
  it('gives every preset a tile of its own', () => {
    const one = shown([
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

  // A person is as far through their day as their fullest budget says.
  it('stands as far through as the fuller of the two budgets', () => {
    const one = shown([preset({ answered: 11, took: 15 })])

    expect(one.find('.presets__done').text()).toBe('75%')
  })

  it('stands on the cards alone where the preset keeps no budget in time', () => {
    const one = shown([
      preset({
        settings: settings({ minutesADay: 0 }),
        budget: { new: 10, reviews: 45, minutes: 0 },
        answered: 11,
        took: 40,
      }),
    ])

    expect(one.text()).toContain('no budget in time')
    expect(one.find('.presets__done').text()).toBe('20%')
  })

  it('is all of the day where a budget has been met exactly', () => {
    const one = shown([preset({ answered: 55, took: 20 })])

    expect(one.find('.presets__done').text()).toBe('100%')
  })

  // Sixty-six answers against thirteen cards is not a day that is done, and a
  // round number would read as one.
  it('says a day answered past its budget is over it, and never a percentage', () => {
    const one = shown([preset({ budget: { new: 3, reviews: 10, minutes: 20 }, answered: 66 })])
    const said = one.find('.presets__done')

    expect(said.text()).toBe('over budget')
    expect(said.attributes('data-over')).toBe('')
  })

  it('says the same of a day that has run past its minutes', () => {
    const one = shown([preset({ answered: 0, took: 31 })])

    expect(one.find('.presets__done').text()).toBe('over budget')
  })

  // The block says the goal and how far through it the day is, and nothing
  // else.
  it('says nothing of counts, of what closes the day, or of the week', () => {
    const said = shown([
      preset({ path: 'A.md', settings: settings({ lightDays: ['sat'] }) }),
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
    const one = shown([preset({ paused: 'no cards a day' })])

    expect(one.find('.presets__preset--paused').exists()).toBe(true)
    expect(one.find('.presets__goal').text()).toBe('no cards a day')
    expect(one.text()).not.toContain('20 minutes a day')
    expect(one.findAll('.presets__done')).toHaveLength(0)
    expect(one.text()).not.toContain('%')
  })

  // This screen is what a person's day comes to. A preset nothing points at is
  // a file being set up, and the editor's preset tab is where it is read.
  it('leaves out a preset no deck points at', () => {
    const one = shown([
      preset({ decks: [], named: 0, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('Nothing points here')
  })

  it('leaves out a preset whose decks hold no card between them', () => {
    const one = shown([
      preset({ decks: [], named: 2, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets').exists()).toBe(false)
    expect(one.text()).not.toContain('no cards')
  })

  // A preset that schedules nothing today is another matter: its decks are in
  // the list saying so, and a person needs to see why.
  it('keeps a preset that schedules nothing today, beside the ones that do', () => {
    const one = shown([
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
    const one = shown([preset({ path: '', name: 'The defaults' })])

    expect(one.text()).toContain('The defaults')
    expect(one.findAll('.presets__done')).toHaveLength(1)
    expect(one.findAll('.presets__settle')).toHaveLength(0)
  })

  it('stands aside where the vault has no preset to show', () => {
    expect(shown([]).find('.presets').exists()).toBe(false)
  })
})

// The figure says how far through the day a person is; the count says what is
// left to do, which is what they are deciding on.
describe('what pressing a preset would ask', () => {
  it('counts it under the figure, and says it is near where minutes govern', () => {
    const one = shown([preset({ cards: 22 })])

    expect(one.find('.presets__left').text()).toBe('about 22 cards')
  })

  // Nothing is estimated where the day is held to counts alone.
  it('counts it exactly where the preset keeps no budget in time', () => {
    const one = shown([
      preset({ cards: 22, budget: { new: 10, reviews: 45, minutes: 0 }, took: 0 }),
    ])

    expect(one.find('.presets__left').text()).toBe('22 cards')
  })

  it('says one card as one', () => {
    expect(shown([preset({ cards: 1 })]).find('.presets__left').text()).toBe('about 1 card')
  })

  it('says nothing of a count where there is nothing to ask', () => {
    expect(shown([preset({ cards: 0 })]).findAll('.presets__left')).toHaveLength(0)
  })

  it('says nothing of a count where the preset schedules nothing today', () => {
    const one = shown([preset({ cards: 0, paused: 'no cards a day' })])

    expect(one.findAll('.presets__left')).toHaveLength(0)
  })
})

describe('sitting down to a preset', () => {
  /** Which tiles can be pressed, in the order they stand. */
  const pressable = (one: ReturnType<typeof shown>): readonly boolean[] =>
    one.findAll('.presets__preset').map((tile) => tile.attributes('disabled') === undefined)

  it('is offered by a preset with cards to ask, and names the preset pressed', async () => {
    const one = shown([preset({ path: 'Sanskrit.md' }), preset({ path: 'Pali.md', name: 'Pali' })])
    const tiles = one.findAll('.presets__preset')

    expect(pressable(one)).toStrictEqual([true, true])

    await tiles[1]?.trigger('click')

    expect(one.emitted('start')).toStrictEqual([['Pali.md']])
  })

  // The decks naming no preset are sat down to the way the rest are, and the
  // note they stand under is no note at all.
  it('names the defaults by the nothing they stand in', async () => {
    const one = shown([preset({ path: '', name: 'The defaults' })])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toStrictEqual([['']])
  })

  // A preset with nothing to offer has said above why, and is not pressed.
  it('offers no sitting where the day under it is already done', async () => {
    const one = shown([preset({ cards: 0 })])

    expect(pressable(one)).toStrictEqual([false])

    await one.find('.presets__preset').trigger('click')

    expect(one.emitted('start')).toBeUndefined()
  })

  it('offers no sitting where the preset schedules nothing today', () => {
    expect(pressable(shown([preset({ cards: 0, paused: 'no cards a day' })]))).toStrictEqual([
      false,
    ])
  })
})
