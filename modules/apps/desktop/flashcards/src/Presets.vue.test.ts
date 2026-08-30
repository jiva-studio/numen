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

  it('draws one meter and no more', () => {
    const meters = shown([preset()]).findAll('.presets__track')

    expect(meters).toHaveLength(1)
    expect(meters[0]?.attributes('style')).toContain('--filled: 0.2')
  })

  // A person is as far through their day as their fullest budget says.
  it('stands as far through as the fuller of the two budgets', () => {
    const one = shown([preset({ answered: 11, took: 15 })])

    expect(one.find('.presets__done').text()).toBe('75%')
    expect(one.find('.presets__track').attributes('style')).toContain('--filled: 0.75')
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
    expect(one.find('.presets__track').attributes('style')).toContain('--filled: 1')
  })

  // Sixty-six answers against thirteen cards is not a day that is done, and a
  // round number would read as one.
  it('says a day answered past its budget is over it, and never a percentage', () => {
    const one = shown([preset({ budget: { new: 3, reviews: 10, minutes: 20 }, answered: 66 })])
    const said = one.find('.presets__done')

    expect(said.text()).toBe('over budget')
    expect(said.attributes('data-over')).toBe('')
    // The meter is full, because it is.
    expect(one.find('.presets__track').attributes('style')).toContain('--filled: 1')
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

    expect(said).not.toContain('cards')
    expect(said).not.toContain('close the day')
    expect(said).not.toContain('Today:')
    expect(said).not.toContain('light in')
  })

  it('greys a preset that schedules nothing and says why', () => {
    const one = shown([preset({ paused: 'no cards a day' })])

    expect(one.find('.presets__preset--paused').exists()).toBe(true)
    expect(one.text()).toContain('no cards a day')
    expect(one.findAll('.presets__track')).toHaveLength(0)
  })

  // A preset nothing points at schedules nobody, and a meter drawn against it
  // says nothing a person can act on.
  it('says of a preset no deck points at that nothing does', () => {
    const one = shown([
      preset({ decks: [], named: 0, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets__alone').text()).toBe('Nothing points here')
    expect(one.findAll('.presets__track')).toHaveLength(0)
    expect(one.find('.presets__preset').text()).toBe('SanskritNothing points here')
  })

  // A deck holding no cards points at its preset all the same, which is not the
  // same state as a preset nothing points at.
  it('says of a preset whose decks are empty how many they are', () => {
    const one = shown([
      preset({ decks: [], named: 1, faces: 0, settings: null, cards: 0, answered: 0, took: 0 }),
    ])

    expect(one.find('.presets__alone').text()).toBe('One deck, no cards')
    expect(one.findAll('.presets__track')).toHaveLength(0)
  })

  // The decks naming no preset are a row like any other, and it carries no act.
  it('draws the row of the defaults as an ordinary preset', () => {
    const one = shown([preset({ path: '', name: 'The defaults' })])

    expect(one.text()).toContain('The defaults')
    expect(one.findAll('.presets__track')).toHaveLength(1)
    expect(one.findAll('.presets__settle')).toHaveLength(0)
  })

  it('stands aside where the vault has no preset to show', () => {
    expect(shown([]).find('.presets').exists()).toBe(false)
  })
})
