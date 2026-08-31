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
  clears: 0,
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
  overdue: 0,
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
  // Two marks are drawn and both are named: where the person stands, and what
  // is suggested. The old third mark, the value in the file, is gone.
  it('names every mark over the picture and not inside it', () => {
    const { tab } = drawn()
    const names = tab.findAll('.control__label').map((one) => one.text())
    expect(names).toStrictEqual([words.now, words.markName('minutes')])
    expect(tab.findAll('.control__now')).toHaveLength(0)
    expect(tab.get('svg').find('text').exists()).toBe(false)
  })

  // The drop line and the value under it belong to the knob, so both stand
  // where the person put it and nowhere else.
  it('drops its line from the knob, wherever the knob is', async () => {
    const { tab } = drawn()
    const dropAt = () => tab.get('.control__drop').attributes('x1')
    const knobAt = () => tab.get('.control__knob').attributes('cx')
    expect(dropAt()).toBe(knobAt())
    await tab.get('[role="slider"]').trigger('keydown', { key: 'Home' })
    expect(dropAt()).toBe(knobAt())
    await tab.get('[role="slider"]').trigger('keydown', { key: 'End' })
    expect(dropAt()).toBe(knobAt())
  })

  // The mark is named for what it is, and the rule it is found by is prose
  // under the picture rather than a word on the mark.
  it('says the rule the second mark is found by, under the picture', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal }, { goal })
      expect(tab.get('.control__why').text()).toBe(words.markRule(goal))
      expect(tab.findAll('.control__label').map((one) => one.text())).toContain(
        words.markName(goal),
      )
    }
    // Nothing about it is a recommendation, so it is not called one.
    expect(words.markName('minutes')).toBe('time enough')
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(words.markName(goal)).not.toContain('suggest')
    }
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
    expect(tab.text()).toContain('you would remember 88% of what you are asked')
    expect(tab.text()).not.toContain('puts 0 cards')
  })

  // The figure is what a person would recall when a card comes round, said as
  // a percentage and named as an act of remembering.
  it('speaks the share as a percentage, and of remembering', () => {
    expect(words.value('retention', 0.9, '')).toBe('90% remembered')
    expect(words.widthAt('retention', 0.9)).toBe('90%')
    expect(words.costs('retention', 0.9, 48, 14, 0.9)).toContain(
      'Remembering 90% of what you are asked',
    )
    expect(words.costs('minutes', 7, 58, 7, 0.9)).toContain(
      'you would remember 90% of what you are asked',
    )
    expect(words.fieldDetail('retention')).toContain('remember')
    for (const said of [
      words.value('retention', 0.9, ''),
      words.costs('retention', 0.9, 48, 14, 0.9),
      words.fieldDetail('retention'),
    ]) {
      expect(said).not.toContain('0.9')
    }
  })

  // The height, the label over it and the sentence under it are one number:
  // every card the sitting puts in front of the person, new and returning.
  it('names the height, the sentence and the axis in cards of a sitting', () => {
    const { tab } = drawn()
    expect(tab.get('.control__name--y').text()).toBe(words.axisY('minutes'))
    expect(tab.get('.control__name--y').text()).toContain('sitting')
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

  // Each axis is named along the axis it names, with its unit in the name.
  it('names both axes where each axis is, under every goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal }, { goal })
      expect(tab.get('.control__name--y').text()).toBe(words.axisY(goal))
      expect(tab.get('.control__name--x').text()).toBe(words.axisX(goal))
    }
    expect(words.axisY('minutes')).toBe('Cards in a sitting')
    expect(words.axisX('minutes')).toBe('Minutes a day')
    expect(words.axisY('retention')).toBe('Minutes a day')
    expect(words.axisX('retention')).toBe('Retention')
    expect(words.axisY('date')).toBe('Minutes a day')
    expect(words.axisX('date')).toBe('Days from today')
  })

  // The words say which way is better and the numbers say how much, so a
  // height can be read off the picture and a place along it can be told.
  it('carries the ends of the band, against the lines they are the height of', () => {
    const numbers = drawn().tab.findAll('.control__number').map((one) => one.text())
    // The band of the fixture runs from no cards a day to a hundred and twenty.
    expect(numbers).toContain(words.heightAt('minutes', 120))
  })

  // A number the line or a mark stands on is dropped: the axis gives way, and
  // the drawing keeps what it has to say.
  it('drops an axis number the drawing stands on rather than print over it', () => {
    // The fixture's curve leaves the foot at the left edge, where the low
    // number would be set.
    const numbers = drawn().tab.findAll('.control__number').map((one) => one.text())
    expect(numbers).not.toContain(words.heightAt('minutes', 0))
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

  it('carries the value at either end of the range, and no words beside them', () => {
    const ends = drawn().tab.get('.control__ends').text()
    expect(ends).toContain(words.widthAt('minutes', 0))
    expect(ends).toContain(words.widthAt('minutes', 30))
    expect(ends).not.toContain('·')
  })

  // The knob says the value it stands on, so the end under it would say it twice.
  it('leaves the end the knob stands on to the knob', async () => {
    const { tab } = drawn()
    const ends = () => tab.findAll('.control__ends span').map((one) => one.text())
    await tab.get('[role="slider"]').trigger('keydown', { key: 'Home' })
    expect(ends()).toStrictEqual(['', words.widthAt('minutes', 30)])
    await tab.get('[role="slider"]').trigger('keydown', { key: 'End' })
    expect(ends()).toStrictEqual([words.widthAt('minutes', 0), ''])
  })

  it('carries the value at the knob, and it follows the knob', async () => {
    const { tab } = drawn()
    const at = () => tab.get('.control__number--knob')
    expect(at().text()).toBe(words.widthAt('minutes', 20))
    await tab.get('[role="slider"]').trigger('keydown', { key: 'End' })
    expect(at().text()).toBe(words.widthAt('minutes', 30))
  })

  it('reads the numbers of each goal in that goal’s own units', () => {
    expect(words.widthAt('retention', 0.8)).toBe('80%')
    expect(words.widthAt('date', 12)).toBe('12 d')
    expect(words.heightAt('date', 45)).toBe('45 min')
    expect(words.heightAt('minutes', 80)).toBe('80 cards')
  })

  // How far behind, and what this pace does about it. The user's own vault:
  // 45 overdue of 160 faces, cleared in 5 days at the place the knob stands at.
  it('says how far behind the preset is and how long this pace takes to clear it', () => {
    const { tab } = drawn({
      overdue: 45,
      cards: 160,
      at: [point(), point(), point({ reviews: 80, clears: 5 }), point({ clears: 2 })],
    })
    expect(tab.text()).toContain('45 of 160 card faces are overdue')
    expect(tab.text()).toContain('nothing is overdue after 5 days')
  })

  it('says a pace that never gets there does not, rather than naming a day', () => {
    const { tab } = drawn({
      overdue: 45,
      cards: 160,
      at: [point(), point(), point({ reviews: 80, clears: -1 }), point({ clears: 2 })],
    })
    expect(tab.text()).toContain('the backlog never clears')
    expect(tab.text()).not.toContain('after -1')
  })

  // Nothing overdue is nothing to clear, and a sentence saying so is noise.
  it('says nothing at all about clearing where nothing is overdue', () => {
    const { tab } = drawn({ overdue: 0, cards: 160 })
    expect(tab.text()).not.toContain('overdue')
    expect(tab.text()).not.toContain('clear')
  })

  // Overdue is the backlog alone. What a sitting offers is that and today's
  // cards together, which is the deck screen's number and a larger one.
  it('says overdue only of the backlog, and never of the whole', () => {
    const said = words.behind(45, 160)
    expect(said).toContain('45')
    expect(said).toContain('160')
    expect(said).not.toContain('due today')
    expect(said).not.toContain('unbegun')
    expect(words.behind(1, 160)).toBe('1 of 160 card faces is overdue.')
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
    overdue: 0,
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
