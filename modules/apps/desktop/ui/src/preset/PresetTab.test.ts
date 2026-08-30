/**
 * The preset tab drawn, in a document.
 *
 * What is asked here is that the curve is the control — one stop on the way
 * round the screen, walked by the arrow keys and written once the key is let
 * go of — and that a field typed by hand stops following the goal and offers
 * the way back.
 */
import { describe, expect, it } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'

import PresetTab from './PresetTab.vue'
import { DEFAULTS, NOWHERE, type Curve, type Point, type Settings } from './core'
import type { Field } from './curve'
import type { Held } from './kind'
import { WORDS as words } from './words'

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  met: true,
  ...over,
})

const curve = (over: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [
    point(),
    point({ reviews: 40, retained: 0.8 }),
    point({ reviews: 80, retained: 0.88 }),
    point({ reviews: 120, retained: 0.93 }),
  ],
  now: { at: 2, value: 20, day: '' },
  suggested: { at: 3, value: 30, day: '' },
  decks: 1,
  honest: true,
  ...over,
})

/** A tab standing at those settings, and everything it was asked to do. */
const standing = (over: Partial<Curve> = {}, settings: Partial<Settings> = {}) => {
  const done: string[] = []
  const byHand = shallowRef<ReadonlySet<Field>>(new Set())
  const place = ref(2)
  const held: Held = {
    id: 'Sanskrit.md',
    settings: () => ({ ...DEFAULTS, ...settings }),
    curve: () => curve(over),
    place: () => place.value,
    byHand: () => byHand.value,
    problems: () => [],
    saying: () => '',
    changed: () => false,
    again: () => void done.push('again'),
    chooses: (goal) => void done.push(`chooses ${goal}`),
    moves: (at) => {
      place.value = at
      done.push(`moves ${at}`)
    },
    settles: () => void done.push('settles'),
    types: (field, value) => {
      byHand.value = new Set([...byHand.value, field])
      done.push(`types ${field} ${value}`)
    },
    follows: (field) => {
      const rest = new Set(byHand.value)
      rest.delete(field)
      byHand.value = rest
      done.push(`follows ${field}`)
    },
    shuts: () => {},
  }
  return { held, done, byHand }
}

const drawn = (over: Partial<Curve> = {}, settings: Partial<Settings> = {}) => {
  const one = standing(over, settings)
  return { ...one, tab: mount(PresetTab, { props: { held: one.held } }) }
}

describe('the one control', () => {
  it('is the curve itself, and one stop on the way round the screen', () => {
    const { tab } = drawn()
    const control = tab.get('[role="slider"]')
    expect(control.attributes('tabindex')).toBe('0')
    expect(control.attributes('aria-valuemin')).toBe('0')
    expect(control.attributes('aria-valuemax')).toBe('30')
    expect(control.attributes('aria-valuenow')).toBe('20')
    expect(control.attributes('aria-valuetext')).toBe(words.value('minutes', 20, ''))
  })

  it('walks the grid a place at a time, and writes once the key is let go of', async () => {
    const { tab, done } = drawn()
    const control = tab.get('[role="slider"]')
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keyup', { key: 'ArrowRight' })
    expect(done).toStrictEqual(['moves 3', 'settles'])
  })

  it('walks to either end, and no further', async () => {
    const { tab, done } = drawn()
    const control = tab.get('[role="slider"]')
    await control.trigger('keydown', { key: 'End' })
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keydown', { key: 'Home' })
    await control.trigger('keydown', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['moves 3', 'moves 3', 'moves 0', 'moves 0'])
  })

  it('leaves a keystroke that is nobody’s to the window', async () => {
    const { tab, done } = drawn()
    await tab.get('[role="slider"]').trigger('keydown', { key: 'k' })
    expect(done).toStrictEqual([])
  })
})

describe('what the control stands at', () => {
  it('is read out in the units of its goal, with what it costs beside it', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain(words.value('minutes', 20, ''))
    expect(tab.text()).toContain(words.costs('minutes', 20, 80, 0, 0.88))
  })

  // A point read as zero draws a screen of zeroes, which reads as a broken one.
  it('reads the cards and the share off the place the knob stands at', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain('holds 80 cards')
    expect(tab.text()).toContain('0.88 of it comes back')
    expect(tab.text()).not.toContain('holds 0 cards')
    expect(tab.text()).not.toContain('0.00 of it comes back')
  })

  it('carries the word for a figure the window guessed, until the answer lands', () => {
    expect(drawn({ honest: false }).tab.text()).toContain(words.about)
    expect(drawn().tab.text()).not.toContain(words.about)
  })

  it('shows the arithmetic of a goal of a date with the sum already done', () => {
    const dated = drawn(
      {
        goal: 'date',
        grid: [10, 20, 30, 40],
        days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
        at: [
          point({ minutes: 90, owed: 400, through: 0.4, met: false }),
          point({ minutes: 60, owed: 200, through: 0.6, met: false }),
          point({ minutes: 40, owed: 80, through: 0.8, met: false }),
          point({ minutes: 30, owed: 0, through: 1, met: true }),
        ],
      },
      { goal: 'date', byDate: '2026-09-29' },
    )
    expect(dated.tab.text()).toContain(words.owing(80))
    expect(dated.tab.text()).toContain(words.unmet)
    // Nothing is refused: the control is there to be moved.
    expect(dated.tab.get('[role="slider"]').attributes('aria-disabled')).toBeUndefined()
  })

  it('says a preset past the day it aimed at has spent its budget', () => {
    const gone = drawn({}, { goal: 'date', byDate: '2000-01-01' })
    expect(gone.tab.text()).toContain(words.spent)
  })
})

describe('a goal with nothing to work on', () => {
  const nothing: Partial<Curve> = {
    at: [point(), point(), point(), point()],
    now: NOWHERE,
    suggested: NOWHERE,
    decks: 0,
  }

  it('says no deck points here, and draws no curve and no figures of nothing', () => {
    const { tab } = drawn(nothing)
    expect(tab.text()).toContain(words.unpointed)
    expect(tab.findAll('[role="slider"]')).toHaveLength(0)
    expect(tab.text()).not.toContain('holds 0 cards')
  })

  it('says the decks pointing here hold no cards, where they do point here', () => {
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).toContain(words.noCards(1))
    expect(drawn({ ...nothing, decks: 3 }).tab.text()).toContain(words.noCards(3))
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).not.toContain(words.unpointed)
  })

  it('leaves its settings there to be set up before a deck points here', () => {
    const { tab, done } = drawn(nothing)
    expect(tab.findAll('.preset__row')).toHaveLength(7)
    tab.get('.preset__row input').setValue('7')
    expect(done).toStrictEqual(['types newADay 7'])
  })
})

describe('the settings the chosen goal schedules by', () => {
  it('leaves the day out where it steers nothing', () => {
    const minutes = drawn().tab
    expect(minutes.findAll('.preset__day')).toHaveLength(0)
    expect(minutes.findAll('.preset__row')).toHaveLength(7)
    const share = drawn({ goal: 'retention' }, { goal: 'retention' }).tab
    expect(share.findAll('.preset__day')).toHaveLength(0)
    expect(share.findAll('.preset__row')).toHaveLength(7)
  })

  it('draws the day under the goal that steers it, and moves the knob by it', async () => {
    const { tab, done } = drawn(
      {
        goal: 'date',
        grid: [10, 20, 30, 40],
        days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
      },
      { goal: 'date', byDate: '2026-09-29' },
    )
    expect(tab.text()).toContain(words.fieldName('byDate'))
    expect(tab.findAll('.preset__row')).toHaveLength(8)
    const day = tab.get('.preset__day')
    expect((day.element as HTMLInputElement).value).toBe('2026-09-29')
    await day.setValue('2026-10-09')
    expect(done).toStrictEqual(['types byDate 2026-10-09'])
  })
})

describe('the settings under the control', () => {
  it('draws a row for each, with what it is beside it', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain(words.fieldName('newADay'))
    expect(tab.text()).toContain(words.fieldDetail('reviewsADay'))
    expect(tab.findAll('.preset__row')).toHaveLength(7)
  })

  // What a budget is spent on is a row like any other, and a person moving it
  // writes the group the way every other row does.
  it('offers the two things a budget is spent on', async () => {
    const { tab, done } = drawn()
    expect(tab.text()).toContain(words.fieldName('counts'))

    const shows = tab
      .findAll('button')
      .find((one) => one.text() === words.countsName('shows'))
    await shows?.trigger('click')
    expect(done).toStrictEqual(['types counts shows'])
  })

  it('marks a row a person typed themselves, and offers it back under the goal', async () => {
    const { tab, done, byHand } = drawn()
    byHand.value = new Set(['reviewsADay'])
    await tab.vm.$nextTick()
    const mine = tab.findAll('.preset__row--mine')
    expect(mine).toHaveLength(1)
    await mine[0]!.get('button').trigger('click')
    expect(done).toStrictEqual(['follows reviewsADay'])
  })

  it('offers no way back under a goal that does not produce that field', async () => {
    const { tab, byHand } = drawn()
    byHand.value = new Set(['newADay'])
    await tab.vm.$nextTick()
    expect(tab.findAll('.preset__row--mine')[0]!.findAll('button')).toHaveLength(0)
  })
})
