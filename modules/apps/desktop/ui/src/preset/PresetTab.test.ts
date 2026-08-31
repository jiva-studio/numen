/**
 * The preset tab drawn, in a document.
 *
 * What is asked here is that the curve is the control — one stop on the way
 * round the screen, walked by the arrow keys and written once the key is let
 * go of — and that a goal draws the settings it schedules by and no others.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
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
  cards: 400,
  honest: true,
  ...over,
})

/** A tab standing at those settings, and everything it was asked to do. */
const standing = (over: Partial<Curve> = {}, settings: Partial<Settings> = {}) => {
  const done: string[] = []
  const place = ref(2)
  const held: Held = {
    id: 'Sanskrit.md',
    settings: () => ({ ...DEFAULTS, ...settings }),
    curve: () => curve(over),
    place: () => place.value,
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
    types: (field, value) => void done.push(`types ${field} ${value}`),
    shuts: () => {},
  }
  return { held, done }
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

  it('offers the three goals by the value each steers, under a label saying so', () => {
    const { tab } = drawn()
    expect(tab.get('.preset__label').text()).toBe(words.goal)
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(tab.text()).toContain(words.goalName(goal))
    }
  })

  // A name inside the picture is scaled with it and is set at no step of the
  // page's type.
  it('names a mark over the picture and not inside it', () => {
    const { tab } = drawn()
    expect(tab.get('.control__label').text()).toBe(words.suggested)
    expect(tab.get('svg').find('text').exists()).toBe(false)
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
    expect(tab.text()).toContain('puts 80 cards in front of you')
    expect(tab.text()).toContain('0.88 of the material comes back')
    expect(tab.text()).not.toContain('puts 0 cards')
    expect(tab.text()).not.toContain('0.00 of the material')
  })

  // The height, the label over it and the sentence under it are one number:
  // every card the sitting puts in front of the person, new and returning.
  it('names the height, the sentence and the axis in cards of a sitting', () => {
    const { tab } = drawn()
    expect(tab.get('.control__height').text()).toBe(words.height('minutes'))
    expect(tab.get('.control__height').text()).toContain('sitting')
    expect(tab.text()).toContain(words.heightAt('minutes', 120))
    expect(tab.text()).toContain(words.costs('minutes', 20, 80, 0, 0.88))
  })

  it('carries the word for a figure the window guessed, until the answer lands', () => {
    expect(drawn({ honest: false }).tab.text()).toContain(words.about)
    expect(drawn().tab.text()).not.toContain(words.about)
  })

  // A line drawn before the answer has to move when it lands, and a picture
  // that moves reads as a glitch. Nothing is drawn until there is an answer.
  it('draws no line and no control while the curve is being worked out', () => {
    const { tab } = drawn({ honest: false })
    expect(tab.findAll('.control__waiting')).toHaveLength(1)
    expect(tab.text()).toContain(words.waiting)
    expect(tab.findAll('svg')).toHaveLength(0)
    expect(tab.findAll('[role="slider"]')).toHaveLength(0)
    expect(tab.findAll('.control__number')).toHaveLength(0)
    expect(tab.get('.control__ends').text()).toBe('')
  })

  it('draws the picture and nothing waiting once the answer has landed', () => {
    const { tab } = drawn()
    expect(tab.findAll('.control__waiting')).toHaveLength(0)
    expect(tab.text()).not.toContain(words.waiting)
    expect(tab.findAll('[role="slider"]')).toHaveLength(1)
  })

  // The rows under the picture keep their room, so the answer landing moves
  // nothing below the plot.
  it('keeps the rows under the picture whether or not the answer has landed', () => {
    for (const one of [drawn({ honest: false }), drawn()]) {
      expect(one.tab.findAll('.control__under')).toHaveLength(1)
      expect(one.tab.findAll('.control__ends')).toHaveLength(1)
    }
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
    // Four lines at most, and none of them the figure or the sentence again.
    expect(dated.tab.findAll('.preset__sums li').length).toBeLessThanOrEqual(4)
    expect(dated.tab.findAll('.preset__sums li').map((one) => one.text())).not.toContain(
      words.value('date', 30, '2026-09-29'),
    )
    // Nothing is refused: the control is there to be moved.
    expect(dated.tab.get('[role="slider"]').attributes('aria-disabled')).toBeUndefined()
  })

  // A flat line is the truth where the counts close the day, and a flat line
  // nobody can read is not an answer.
  it('says what closes a day a longer one buys nothing on', () => {
    const shut = drawn(
      { at: [point({ reviews: 13 }), point({ reviews: 13 }), point({ reviews: 13 })], grid: [0, 10, 20] },
      { newADay: 8, reviewsADay: 5 },
    )
    expect(shut.tab.text()).toContain(words.closed(8, 5))
    expect(drawn().tab.text()).not.toContain(words.closed(10, 200))
  })

  it('names what the height of the picture is read in', () => {
    expect(drawn().tab.text()).toContain(words.height('minutes'))
    expect(drawn({ goal: 'retention' }).tab.text()).toContain(words.height('retention'))
  })

  // The words say which way is better and the numbers say how much, so a
  // height can be read off the picture and a place along it can be told.
  it('carries the two ends of the band, against the lines they are the height of', () => {
    const numbers = drawn().tab.findAll('.control__number').map((one) => one.text())
    // The band of the fixture runs from no cards a day to a hundred and twenty.
    expect(numbers).toContain(words.heightAt('minutes', 120))
    expect(numbers).toContain(words.heightAt('minutes', 0))
  })

  // Two ends of a band of no width are one number, and one number said twice
  // says nothing.
  it('says the one value once where the curve never moves', () => {
    const flat = drawn({ at: [point({ reviews: 2 }), point({ reviews: 2 }), point({ reviews: 2 })] })
    const numbers = flat.tab.findAll('.control__number:not(.control__number--knob)')
    expect(numbers.map((one) => one.text())).toStrictEqual([words.heightAt('minutes', 2)])
  })

  // The knob's value rides a line of its own, so a knob at either end cannot
  // print over a number read off the picture.
  it('keeps the knob’s value on its own line, clear of the picture’s numbers', async () => {
    const { tab } = drawn()
    const over = tab.get('.control__over')
    const under = tab.get('.control__under')
    expect(under.findAll('.control__number--knob')).toHaveLength(1)
    expect(over.findAll('.control__number--knob')).toHaveLength(0)

    // At either end the knob's value is pulled back inside the picture's width.
    const slider = tab.get('[role="slider"]')
    await slider.trigger('keydown', { key: 'Home' })
    expect(tab.get('.control__number--knob').attributes('style')).toContain('translate: 0 0')
    await slider.trigger('keydown', { key: 'End' })
    expect(tab.get('.control__number--knob').attributes('style')).toContain('translate: -100% 0')
  })

  it('carries the value at either end of the range, beside the words there', () => {
    const ends = drawn().tab.get('.control__ends').text()
    expect(ends).toContain(words.ends('minutes')[0])
    expect(ends).toContain(words.widthAt('minutes', 0))
    expect(ends).toContain(words.ends('minutes')[1])
    expect(ends).toContain(words.widthAt('minutes', 30))
  })

  it('carries the value at the knob, and it follows the knob', async () => {
    const { tab } = drawn()
    const at = () => tab.get('.control__number--knob')
    expect(at().text()).toBe(words.widthAt('minutes', 20))
    await tab.get('[role="slider"]').trigger('keydown', { key: 'End' })
    expect(at().text()).toBe(words.widthAt('minutes', 30))
  })

  it('reads the numbers of each goal in that goal’s own units', () => {
    expect(words.widthAt('retention', 0.8)).toBe('0.80')
    expect(words.widthAt('date', 12)).toBe('12 d')
    expect(words.heightAt('date', 45)).toBe('45 min')
    expect(words.heightAt('minutes', 80)).toBe('80 cards')
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
    cards: 0,
  }

  it('says no deck points here, and draws no curve and no figures of nothing', () => {
    const { tab } = drawn(nothing)
    expect(tab.text()).toContain(words.unpointed)
    expect(tab.findAll('[role="slider"]')).toHaveLength(0)
    expect(tab.text()).not.toContain('holds 0 cards')
  })

  it('says the decks pointing here hold no cards, where they do point here', () => {
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).toContain(words.noCards(1))
    expect(drawn({ ...nothing, decks: 4 }).tab.text()).toContain(words.noCards(4))
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).not.toContain(words.unpointed)
  })

  // A preset holding cards is never told it holds none, so a curve of zeros
  // keeps its control and says nothing about the vault.
  it('draws the control for a preset holding cards, whatever its curve comes to', () => {
    const { tab } = drawn({ ...nothing, decks: 4, cards: 900 })
    expect(tab.findAll('[role="slider"]')).toHaveLength(1)
    expect(tab.text()).not.toContain(words.unpointed)
    expect(tab.text()).not.toContain(words.noCards(4))
  })

  it('draws it under a goal of a date, where a curve of zeros is likeliest', () => {
    const { tab } = drawn({
      ...nothing,
      decks: 4,
      cards: 900,
      goal: 'date',
      grid: [1, 2, 3, 4],
      days: ['2026-09-01', '2026-09-02', '2026-09-03', '2026-09-04'],
    })
    expect(tab.findAll('[role="slider"]')).toHaveLength(1)
  })

  it('leaves its settings there to be set up before a deck points here', () => {
    const { tab, done } = drawn(nothing)
    expect(tab.findAll('.preset__row')).toHaveLength(4)
    tab.get('.preset__row input').setValue('7')
    expect(done).toStrictEqual(['types minutesADay 7'])
  })
})

describe('the settings the chosen goal schedules by', () => {
  /** What each row of the receipt is called, which is what the goal draws. */
  const rows = (tab: ReturnType<typeof mount>) =>
    tab.findAll('.preset__row .preset__name').map((one) => one.text())

  // A goal names one budget. The budgets of the other two are not drawn, so
  // nothing on the screen offers to close a day by a measure nobody named.
  it('draws the minutes alone under a goal of minutes', () => {
    const { tab } = drawn()
    expect(rows(tab)).toStrictEqual([
      words.fieldName('minutesADay'),
      words.fieldName('counts'),
      words.fieldName('lightDays'),
      words.fieldName('evenLoad'),
    ])
    expect(tab.findAll('.preset__day')).toHaveLength(0)
  })

  it('draws the target and the counts that close a day under a goal of retention', () => {
    const { tab } = drawn({ goal: 'retention' }, { goal: 'retention' })
    expect(rows(tab)).toStrictEqual([
      words.fieldName('newADay'),
      words.fieldName('reviewsADay'),
      words.fieldName('retention'),
      words.fieldName('counts'),
      words.fieldName('lightDays'),
      words.fieldName('evenLoad'),
    ])
    expect(tab.findAll('.preset__day')).toHaveLength(0)
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
    expect(rows(tab)).toStrictEqual([
      words.fieldName('byDate'),
      words.fieldName('counts'),
      words.fieldName('lightDays'),
      words.fieldName('evenLoad'),
    ])
    const day = tab.get('.preset__day')
    expect((day.element as HTMLInputElement).value).toBe('2026-09-29')
    await day.setValue('2026-10-09')
    expect(done).toStrictEqual(['types byDate 2026-10-09'])
  })
})

describe('the settings under the control', () => {
  it('draws a row for each, with what it is beside it', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain(words.fieldName('minutesADay'))
    expect(tab.text()).toContain(words.fieldDetail('minutesADay'))
    expect(tab.text()).toContain(words.fieldName('counts'))
    expect(tab.findAll('.preset__row')).toHaveLength(4)
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

  // Every value on the screen is the person's own, so no row is marked and
  // none is offered back to anything.
  it('draws every row alike, with nothing offered back to the goal', () => {
    const { tab } = drawn()
    expect(tab.findAll('.preset__row .preset__answer')).toHaveLength(0)
  })
})
