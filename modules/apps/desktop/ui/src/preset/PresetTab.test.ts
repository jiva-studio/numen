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
  closed: '',
  clears: 0,
  backlog: [],
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
  unbegun: 0,
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
  // The knob is where the person put it and needs no telling. The other mark
  // is not obvious and carries its name.
  it('names the mark that needs a name, and leaves the knob unnamed', () => {
    const { tab } = drawn()
    const names = tab.findAll('.control__label').map((one) => one.text())
    expect(names).toStrictEqual([words.markName('minutes')])
    expect(tab.text()).not.toContain('you are here')
    expect(tab.get('svg').find('text').exists()).toBe(false)
  })

  // A person moving the control needs to know what it is acting on, which the
  // picture itself never says.
  // A readout of what is being steered, scanned and not read: each figure its
  // own tile, with the word for what it counts under it.
  it('says what the control is acting on, as tiles over the picture', () => {
    const { tab } = drawn({ decks: 4, cards: 160, overdue: 45, unbegun: 30 })
    const tiles = tab
      .findAll('.control__tile')
      .map((one) => [one.get('.control__figure').text(), one.get('.control__word').text()])
    expect(tiles).toStrictEqual([
      ['4', 'decks'],
      ['160', 'cards'],
      ['45', 'overdue'],
      ['30', 'new'],
    ])
  })

  // The tiles take an equal share of the width, so a figure standing at
  // nothing leaves the rest to spread over it.
  it('leaves out a tile whose figure stands at nothing', () => {
    const { tab } = drawn({ decks: 4, cards: 160, overdue: 0, unbegun: 0 })
    expect(tab.findAll('.control__tile')).toHaveLength(2)
  })

  // A card nobody has answered is new, which is what the rest of the product
  // calls it. Overdue keeps its own word, which is a distinction of its own.
  it('calls a card nobody has answered new', () => {
    const said = words.material(4, 160, 45, 30).map((one) => one.name)
    expect(said).toContain('new')
    expect(said).not.toContain('unbegun')
    expect(said).toContain('overdue')
  })

  it('leaves out whichever figure stands at nothing', () => {
    const names = (over: number, fresh: number) =>
      words.material(1, 160, over, fresh).map((one) => `${one.figure} ${one.name}`)
    expect(names(0, 30)).toStrictEqual(['1 deck', '160 cards', '30 new'])
    expect(names(45, 0)).toStrictEqual(['1 deck', '160 cards', '45 overdue'])
    expect(names(0, 0)).toStrictEqual(['1 deck', '160 cards'])
  })

  // A ring round the whole picture reads as a frame. Focus belongs on the one
  // thing the keyboard moves.
  it('shows focus on the knob and draws no ring round the picture', () => {
    const style = drawn().tab.get('[role="slider"]').attributes('class')
    expect(style).toContain('control__picture')
    expect(drawn().tab.get('[role="slider"]').attributes('tabindex')).toBe('0')
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

  // The mark keeps its short name on the picture, and the paragraph says the
  // value it stands at and what that figure means. Nothing is captioned.
  it('says what the second mark stands at and means, in the paragraph', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal }, { goal })
      expect(tab.findAll('.control__why')).toHaveLength(0)
      expect(tab.get('.preset__costing').text()).toContain(words.markMeans(goal, 30))
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

  // The value first and the meaning after it, so the figure is what is read.
  it('names the mark’s own value before saying what it is', () => {
    expect(words.markMeans('minutes', 26)).toBe(
      '26 minutes a day is the shortest day the clock no longer cuts short.',
    )
    expect(words.markMeans('date', 45)).toBe(
      "45 days off is the first day this preset's own budget gets through the material.",
    )
    expect(words.markMeans('retention', 0.85)).toContain('85% remembered is the target')
  })

  // What the wire says is what there is: a curve answering no suggestion draws
  // no dot, carries no name and explains nothing.
  it('draws and says nothing where the curve answers no suggestion', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal, suggested: NOWHERE }, { goal })
      expect(tab.findAll('.control__suggested')).toHaveLength(0)
      expect(tab.findAll('.control__label')).toHaveLength(0)
      expect(tab.text()).not.toContain(words.markName(goal))
      expect(tab.get('.preset__costing').text()).not.toContain(words.markMeans(goal, 30))
    }
    // The knob and its number are the person's own and stand either way.
    expect(drawn({ suggested: NOWHERE }).tab.findAll('.control__knob')).toHaveLength(1)
    expect(drawn({ suggested: NOWHERE }).tab.findAll('.control__number--knob')).toHaveLength(1)
  })
})

describe('the band of what stands overdue', () => {
  /** A curve whose place the knob stands at carries a backlog that climbs. */
  const climbing = (backlog: readonly number[] = [16, 21, 55, 66, 65, 78]) =>
    drawn({
      at: [point(), point(), point({ reviews: 80, backlog }), point()],
    })

  it('is a plot of its own under the picture, drawn over the days ahead', () => {
    const { tab } = climbing()
    expect(tab.findAll('.control__backlog')).toHaveLength(1)
    expect(tab.get('.control__backlog').attributes('d')?.startsWith('M')).toBe(true)
  })

  // A day too short to carry what falls due adds to the pile, and the climb is
  // what the picture is for. The band is scaled to this one place's own run.
  it('draws a backlog that climbs as climbing, and not flat', () => {
    const heights = (d: string) => d.split(/[ML]/).slice(1).map((one) => Number(one.split(' ')[1]))
    const drawnAt = heights(climbing().tab.get('.control__backlog').attributes('d') ?? '')
    expect(drawnAt[0]).toBeGreaterThan(drawnAt[5] ?? 0)
    expect(drawnAt[2]).toBeLessThan(drawnAt[1] ?? 0)
    expect(new Set(drawnAt).size).toBeGreaterThan(4)
  })

  // Nothing overdue is the foot, so the height is read against a floor that
  // means something.
  it('reads from nothing overdue to the most this pace ever stands at', () => {
    const said = climbing()
      .tab.findAll('.control__number')
      .map((one) => one.text())
    expect(said).toContain(words.backlogHeightAt(78))
    // A number the line stands on gives way to it, so the foot is read off a
    // run that leaves the left edge of the band clear.
    const falling = climbing([78, 60, 40, 20, 5, 0])
      .tab.findAll('.control__number')
      .map((one) => one.text())
    expect(falling).toContain(words.backlogHeightAt(0))
  })

  it('names both its axes, in the same voice as the picture over it', () => {
    const { tab } = climbing()
    const names = tab.findAll('.control__name--y').map((one) => one.text())
    expect(names).toStrictEqual([words.axisY('minutes'), words.backlogY])
    expect(tab.findAll('.control__name--x').map((one) => one.text())).toContain(words.backlogX)
  })

  it('carries the days at either end, which the goal’s grid says nothing about', () => {
    const { tab } = climbing()
    const ends = tab.findAll('.control__ends').map((one) => one.text())
    expect(ends[1]).toContain(words.backlogWidthAt(6))
  })

  // The band is read and never dragged, so it is no stop on the way round the
  // screen and offers nothing to the keyboard.
  it('is read and not dragged, so the one control stays the one control', () => {
    const { tab } = climbing()
    expect(tab.findAll('[role="slider"]')).toHaveLength(1)
    expect(tab.findAll('svg')).toHaveLength(2)
  })

  // The page keeps its height whether or not there is a backlog to draw.
  it('keeps its room where the place the knob stands carries no backlog', () => {
    const { tab } = drawn()
    expect(tab.findAll('.control__backlog')).toHaveLength(0)
    expect(tab.findAll('.control__room')).toHaveLength(2)
    expect(tab.findAll('.control__name--y').map((one) => one.text())).toContain(words.backlogY)
  })

  // The room a plot is drawn in is the same box in every state it has, so
  // nothing under the picture moves when the answer lands.
  it('holds one room for the plot, waiting, drawn and empty alike', () => {
    const room = (over: Partial<Curve> = {}) =>
      drawn(over)
        .tab.findAll('.control__room')
        .map((one) => one.attributes('style'))
    expect(room({ honest: false })).toStrictEqual(room())
    expect(room({ at: [point(), point(), point(), point()] })).toStrictEqual(room())
  })

  // The backlog is read off the place the knob stands at, so walking the grid
  // walks the band with it.
  it('follows the knob, since each place of the grid keeps its own', async () => {
    const { tab } = drawn({
      at: [
        point({ backlog: [1, 2, 3] }),
        point({ backlog: [9, 9, 9] }),
        point({ backlog: [4, 3, 2] }),
        point({ backlog: [8, 4, 0] }),
      ],
    })
    const line = () => tab.get('.control__backlog').attributes('d')
    const was = line()
    await tab.get('[role="slider"]').trigger('keydown', { key: 'End' })
    expect(line()).not.toBe(was)
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
      // The picture's own row of ends, and the band's under it.
      expect(one.tab.findAll('.control__ends')).toHaveLength(2)
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
    // One telling, not a list: the whole of it is two paragraphs at most, and
    // the arithmetic is joined into sentences rather than set line by line.
    expect(dated.tab.text()).toContain(words.dated(40, 20, 0.8, 80, false))
    expect(dated.tab.findAll('.preset__sums')).toHaveLength(0)
    // However many facts a date carries, they are one block and not several.
    expect(dated.tab.findAll('.preset__costing')).toHaveLength(1)
    // Nothing is refused: the control is there to be moved.
    expect(dated.tab.get('[role="slider"]').attributes('aria-disabled')).toBeUndefined()
  })

  // A flat line is the truth where a count closes the day, and a flat line
  // nobody can read is not an answer. What closed it is on the wire, so it is
  // said wherever the goal on screen is not what closed it.
  it('says what closes the day where the goal on screen does not', () => {
    const shut = drawn(
      {
        grid: [0, 10, 20, 30],
        at: [
          point({ reviews: 13, closed: 'reviews_a_day' }),
          point({ reviews: 13, closed: 'reviews_a_day' }),
          point({ reviews: 13, closed: 'reviews_a_day' }),
          point({ reviews: 13, closed: 'reviews_a_day' }),
        ],
      },
      { newADay: 8, reviewsADay: 5 },
    )
    expect(shut.tab.text()).toContain(words.limiting('minutes', 'reviews_a_day'))
  })

  // A target closes no day of its own, so a curve flat in minutes under that
  // goal is a count doing the limiting and says which one.
  it('names the count that limits a preset worked to a target', () => {
    const shut = drawn(
      {
        goal: 'retention',
        at: [
          point({ minutes: 1, closed: 'reviews_a_day' }),
          point({ minutes: 1, closed: 'reviews_a_day' }),
          point({ minutes: 1, closed: 'reviews_a_day' }),
          point({ minutes: 1, closed: 'reviews_a_day' }),
        ],
      },
      { goal: 'retention', newADay: 12, reviewsADay: 1 },
    )
    const said = shut.tab.get('.preset__costing').text()
    expect(said).toContain(words.limiting('retention', 'reviews_a_day'))
    expect(said).toContain('The target is not what limits this preset here')
    expect(said).toContain('the reviews a day')
  })

  it('says nothing of what closes the day where the goal on screen closes it', () => {
    const { tab } = drawn({
      at: [
        point({ closed: 'minutes_a_day' }),
        point({ closed: 'minutes_a_day' }),
        point({ reviews: 80, closed: 'minutes_a_day' }),
        point({ closed: 'minutes_a_day' }),
      ],
    })
    expect(tab.text()).not.toContain('is not what limits this preset')
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
    const ends = () =>
      tab
        .findAll('.control__ends')[0]
        ?.findAll('span')
        .map((one) => one.text())
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
    // What the sitting comes to and where the backlog stands are one block of
    // prose, not two blocks stacked.
    const said = tab.get('.preset__costing').text()
    expect(said).toContain(words.costs('minutes', 20, 80, 0, 0))
    expect(said).toContain(words.behind(45, 160, 5))
    expect(tab.findAll('.preset__costing')).toHaveLength(1)
  })

  it('says a pace that never gets there does not, rather than naming a day', () => {
    const { tab } = drawn({
      overdue: 45,
      cards: 160,
      at: [point(), point(), point({ reviews: 80, clears: -1 }), point({ clears: 2 })],
    })
    expect(tab.text()).toContain('this pace does not get on top of it')
    expect(tab.text()).not.toContain('after -1')
  })

  // Nothing overdue is nothing to clear, and a sentence saying so is noise.
  it('says nothing at all about clearing where nothing is overdue', () => {
    const { tab } = drawn({ overdue: 0, cards: 160 })
    expect(tab.get('.preset__costing').text()).not.toContain('overdue')
    expect(tab.get('.preset__costing').text()).not.toContain('clear')
  })

  // Overdue is the backlog alone. What a sitting offers is that and today's
  // cards together, which is the deck screen's number and a larger one.
  it('says overdue only of the backlog, and never of the whole', () => {
    const said = words.behind(45, 160, 5)
    expect(said).toContain('45')
    expect(said).toContain('160')
    expect(said).not.toContain('due today')
    expect(said).not.toContain('unbegun')
    expect(words.behind(1, 160, 1)).toBe(
      '1 of 160 cards is overdue, and this pace has nothing overdue after 1 day.',
    )
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
  unbegun: 0,
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
      words.fieldName('backlog'),
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
      words.fieldName('backlog'),
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
      words.fieldName('lightDays'),
      words.fieldName('evenLoad'),
    ])
    expect(rows(tab)).not.toContain(words.fieldName('backlog'))
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
    expect(tab.text()).toContain(words.fieldName('lightDays'))
    expect(tab.findAll('.preset__row')).toHaveLength(4)
  })

  // What a budget is spent on is a row like any other, and a person moving it
  // writes the group the way every other row does. It stands under the goal
  // whose budget is cards.
  it('offers the two things a budget is spent on', async () => {
    const { tab, done } = drawn({ goal: 'retention' }, { goal: 'retention' })
    expect(tab.text()).toContain(words.fieldName('counts'))

    const shows = tab
      .findAll('button')
      .find((one) => one.text() === words.countsName('shows'))
    await shows?.trigger('click')
    expect(done).toStrictEqual(['types counts shows'])
  })

  // A share is chosen by what it does and not typed as a figure, so the row
  // offers the three shares by name.
  it('offers the share of a day the debt takes, by what each share does', async () => {
    const { tab, done } = drawn()
    expect(tab.text()).toContain(words.fieldName('backlog'))
    for (const share of [100, 50, 0]) {
      expect(tab.text()).toContain(words.backlogName(share))
    }
    const first = tab.findAll('button').find((one) => one.text() === words.backlogName(0))
    await first?.trigger('click')
    expect(done).toStrictEqual(['types backlog 0'])
  })

  // A file may carry any share at all, and the one it carries is offered in
  // its place along the scale.
  it('offers a share of its own where the file carries one the three do not', () => {
    const { tab } = drawn({}, { backlog: 30 })
    const said = ['Overdue first', 'Split', '30%', 'New first']
    for (const one of said) expect(tab.text()).toContain(one)
    expect(words.backlogName(30)).toBe('30%')
  })

  // Every value on the screen is the person's own, so no row is marked and
  // none is offered back to anything.
  it('draws every row alike, with nothing offered back to the goal', () => {
    const { tab } = drawn()
    expect(tab.findAll('.preset__row .preset__answer')).toHaveLength(0)
  })
})
