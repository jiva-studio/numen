/**
 * The preset tab drawn, in a document.
 *
 * What is asked here is that the curve is the control — one stop on the way
 * round the screen, walked by the arrow keys and written once the key is let
 * go of — and that a goal draws the settings it schedules by and no others.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'

import PresetTab from './PresetTab.vue'
import { BOUNDS, DEFAULTS, NOWHERE, type Curve, type Point, type Settings } from './core'
import { clearing, type Field } from './curve'
import { FOOT } from './drawing'
import type { Held } from './kind'
import { WORDS as words } from './words'

// The track of a share is measured as it is drawn, and a document with no
// layout in it watches nothing for size.
beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/** The heights a drawn path stands at, in the picture's own units. */
const heights = (d: string): readonly number[] =>
  d
    .split(/[ML]/)
    .slice(1)
    .map((one) => Number(one.trim().split(' ')[1]))

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
  const byHand = shallowRef<ReadonlySet<Field>>(new Set())
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
    types: (field, value) => void done.push(`types ${field} ${value}`),
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
    const control = tab.get('.control__picture[role="slider"]')
    expect(control.attributes('tabindex')).toBe('0')
    expect(control.attributes('aria-valuemin')).toBe('0')
    expect(control.attributes('aria-valuemax')).toBe('30')
    expect(control.attributes('aria-valuenow')).toBe('20')
    expect(control.attributes('aria-valuetext')).toBe(words.value('minutes', 20, ''))
  })

  it('walks the grid a place at a time, and writes once the key is let go of', async () => {
    const { tab, done } = drawn()
    const control = tab.get('.control__picture[role="slider"]')
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keyup', { key: 'ArrowRight' })
    expect(done).toStrictEqual(['moves 3', 'settles'])
  })

  it('walks to either end, and no further', async () => {
    const { tab, done } = drawn()
    const control = tab.get('.control__picture[role="slider"]')
    await control.trigger('keydown', { key: 'End' })
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keydown', { key: 'Home' })
    await control.trigger('keydown', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['moves 3', 'moves 3', 'moves 0', 'moves 0'])
  })

  it('leaves a keystroke that is nobody’s to the window', async () => {
    const { tab, done } = drawn()
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'k' })
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

  // A figure beside an area measures nothing. Each picture carries the two
  // lines its numbers are read against.
  it('draws both axes on both pictures, in the rule the window separates with', () => {
    const { tab } = drawn({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 12, 30] }), point()],
    })
    const rules = tab.findAll('.control__rule')
    expect(rules).toHaveLength(4)
    // Each picture has one line up its left side and one along its foot.
    const upright = rules.filter((one) => one.attributes('x1') === one.attributes('x2'))
    const along = rules.filter((one) => one.attributes('y1') === one.attributes('y2'))
    expect(upright).toHaveLength(2)
    expect(along).toHaveLength(2)
  })

  // The knob shows its own value on the line under the picture and nothing
  // else, so what the curve is telling a person dragging it is said over it:
  // the value being held first, then what that value buys.
  it('says what this place buys, in a bubble over the knob, value first', () => {
    const { tab } = drawn({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 4, 0, 0] }), point()],
    })
    const said = tab
      .get('.control__perch')
      .findAll('.control__bought')
      .map((one) => one.text())
    expect(said).toStrictEqual([
      '20 minutes a day',
      '80 cards a sitting',
      'overdue gone in 3 days',
    ])
    // Its own value on the width stays on its own line under the picture.
    expect(tab.get('.control__number--knob').text()).toBe(words.widthAt('minutes', 20))
    // The tail is on the knob, so the numbers belong to that place and not to
    // the picture as a whole.
    expect(tab.findAll('.control__tail')).toHaveLength(1)
  })

  it('says what a target costs, and a date the days and the day’s length', () => {
    const kept = drawn(
      {
        goal: 'retention',
        grid: [0.7, 0.8, 0.9, 0.95],
        now: { at: 2, value: 0.9, day: '' },
        at: [point(), point(), point({ minutes: 14, backlog: [5, 0] }), point()],
      },
      { goal: 'retention' },
    )
    expect(kept.tab.findAll('.control__bought').map((one) => one.text())).toStrictEqual([
      '90% remembered',
      '14 minutes a day',
      'overdue gone in 2 days',
    ])

    // Under a date every card is to be got through anyway, so when the overdue
    // goes says nothing and no line is drawn for it.
    const dated = drawn(
      {
        goal: 'date',
        grid: [10, 20, 30, 40],
        days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
        now: { at: 2, value: 30, day: '2026-09-29' },
        at: [point(), point(), point({ minutes: 40, backlog: [9, 4, 0, 8] }), point()],
      },
      { goal: 'date', byDate: '2026-09-29' },
    )
    expect(dated.tab.findAll('.control__bought').map((one) => one.text())).toStrictEqual([
      '30 days off',
      '40 minutes a day',
    ])
  })

  // Nothing overdue is nothing to say about it, and a pile the days projected
  // never clear is said in words and not as a figure nobody can stand behind.
  it('draws the overdue line only where there is something overdue', () => {
    const none = drawn({
      at: [point(), point(), point({ reviews: 80, backlog: [0, 0, 0] }), point()],
    })
    expect(none.tab.findAll('.control__bought').map((one) => one.text())).toStrictEqual([
      '20 minutes a day',
      '80 cards a sitting',
    ])

    const never = drawn({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 21, 40] }), point()],
    })
    expect(never.tab.findAll('.control__bought').map((one) => one.text())).toStrictEqual([
      '20 minutes a day',
      '80 cards a sitting',
      'not within 3 days',
    ])
  })

  // A preset's own value need not sit on the grid, and the place it opens at
  // is only the nearest one. The knob does not claim a value it is not at.
  it('reads the preset’s own value where it opens, and the place once walked', async () => {
    const { tab } = drawn({ grid: [0, 10, 20, 30], now: { at: 2, value: 23, day: '' } })
    const first = () => tab.findAll('.control__bought')[0]?.text()
    expect(first()).toBe('23 minutes a day')
    expect(tab.get('.control__number--knob').text()).toBe(words.widthAt('minutes', 23))
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'End' })
    expect(first()).toBe('30 minutes a day')
    expect(tab.get('.control__number--knob').text()).toBe(words.widthAt('minutes', 30))
  })

  // The figure is read off the very run the band is drawn from, so the picture
  // and the words can never disagree.
  it('takes the day it goes off the projection the band is drawn from', () => {
    expect(clearing([9, 4, 0, 0])).toBe(3)
    expect(clearing([9, 21, 40])).toBe(-1)
    expect(clearing([0, 0, 0])).toBe(null)
    expect(clearing([])).toBe(null)
  })

  // A bubble that covered the line would hide the thing it is about.
  it('stands over the knob, and under it where over would leave the picture', async () => {
    const { tab } = drawn()
    // Which way the bubble is lifted off its anchor, which is the anchor the
    // tail sits on.
    const lift = () =>
      /translate:[^;]*\s([^\s;]+);?/.exec(tab.get('.control__perch').attributes('style') ?? '')?.[1]
    const tail = () => tab.get('.control__tail').attributes('class') ?? ''
    const slider = tab.get('.control__picture[role="slider"]')

    // At the foot of this curve there is room above the knob.
    await slider.trigger('keydown', { key: 'Home' })
    expect(lift()).toBe('-100%')
    expect(tail()).not.toContain('control__tail--under')

    // At its top there is none, and the bubble and its tail turn over together.
    await slider.trigger('keydown', { key: 'End' })
    expect(lift()).toBe('0')
    expect(tail()).toContain('control__tail--under')
  })

  // A ring round the whole picture reads as a frame. Focus belongs on the one
  // thing the keyboard moves.
  it('shows focus on the knob and draws no ring round the picture', () => {
    const style = drawn().tab.get('.control__picture[role="slider"]').attributes('class')
    expect(style).toContain('control__picture')
    expect(drawn().tab.get('.control__picture[role="slider"]').attributes('tabindex')).toBe('0')
  })

  // The drop line and the value under it belong to the knob, so both stand
  // where the person put it and nowhere else.
  it('drops its line from the knob, wherever the knob is', async () => {
    const { tab } = drawn()
    const dropAt = () => tab.get('.control__drop').attributes('x1')
    const knobAt = () => tab.get('.control__knob').attributes('cx')
    expect(dropAt()).toBe(knobAt())
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'Home' })
    expect(dropAt()).toBe(knobAt())
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'End' })
    expect(dropAt()).toBe(knobAt())
  })

  // The mark keeps its short name on the picture, and the paragraph says the
  // value it stands at and what that figure means. Nothing is captioned.
  it('draws and says nothing where the curve answers no suggestion', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal, suggested: NOWHERE }, { goal })
      expect(tab.findAll('.control__suggested')).toHaveLength(0)
      expect(tab.findAll('.control__label')).toHaveLength(0)
      expect(tab.text()).not.toContain(words.markName(goal))
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
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(1)
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

  // Nothing overdue is the floor the band is read up from, and a run holding
  // nothing at all lies along it. A line through the middle would read as a
  // pile standing at something.
  it('lays a run of nothing overdue along the floor, not through the middle', () => {
    const heights = (d: string) =>
      d
        .split(/[ML]/)
        .slice(1)
        .map((one) => Number(one.split(' ')[1]))
    const flat = heights(
      climbing([0, 0, 0, 0, 0, 0]).tab.get('.control__backlog').attributes('d') ?? '',
    )
    const floor = heights(
      climbing([12, 8, 4, 0, 0, 0]).tab.get('.control__backlog').attributes('d') ?? '',
    )
    // Every place of the empty run stands where the run that clears comes to
    // rest, which is the foot the axis is drawn along.
    expect(new Set(flat).size).toBe(1)
    expect(flat[0]).toBe(floor[5])
    expect(flat[0]).toBe(floor[4])
  })

  it('says nothing overdue once, on the floor, where the run holds nothing', () => {
    const said = climbing([0, 0, 0, 0, 0, 0])
      .tab.findAll('.control__number')
      .map((one) => one.text())
    expect(said).toContain(words.backlogHeightAt(0))
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
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'End' })
    expect(line()).not.toBe(was)
  })
})

describe('what the control stands at', () => {
  it('is read out in the units of its goal, with what it buys over the knob', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain(words.value('minutes', 20, ''))
    expect(tab.findAll('.control__bought').map((one) => one.text())).toContain('80 cards a sitting')
  })

  // Nobody reads under the picture while dragging, so what a place buys is
  // said in the bubble and the room under the picture is the axis alone.
  it('leaves the axis and what belongs to it under the picture', () => {
    const { tab } = drawn()
    const foot = tab.findAll('.control__foot')[0]
    expect(foot?.get('.control__number--knob').text()).toBe(words.widthAt('minutes', 20))
    expect(foot?.get('.control__name--x').text()).toBe(words.axisX('minutes'))
    // Nothing under the picture but the figure the page is headed by.
    expect(tab.findAll('.preset__reading')[0]?.text()).toBe(words.value('minutes', 20, ''))
  })

  // A point read as zero draws a screen of zeroes, which reads as a broken one.
  it('reads the cards off the place the knob stands at', () => {
    const { tab } = drawn()
    const said = tab.findAll('.control__bought').map((one) => one.text())
    expect(said).toContain('80 cards a sitting')
    expect(said).not.toContain('0 cards a sitting')
  })

  // The figure is what a person would recall when a card comes round, said as
  // a percentage and named as an act of remembering.
  it('speaks the share as a percentage, and of remembering', () => {
    expect(words.value('retention', 0.9, '')).toBe('90% remembered')
    expect(words.widthAt('retention', 0.9)).toBe('90%')
    expect(words.fieldDetail('retention')).toContain('remember')
    for (const said of [
      words.value('retention', 0.9, ''),
      words.buys('retention', {
        value: 0.9,
        reviews: 48,
        minutes: 14,
        horizon: 0,
        clears: null,
      }).join(' '),
      words.fieldDetail('retention'),
    ]) {
      expect(said).not.toContain('0.9')
    }
  })

  // The height, the bubble over the knob and the axis are one number: every
  // card the sitting puts in front of the person, new and returning.
  it('names the height, the bubble and the axis in cards of a sitting', () => {
    const { tab } = drawn()
    expect(tab.get('.control__name--y').text()).toBe(words.axisY('minutes'))
    expect(tab.get('.control__name--y').text()).toContain('sitting')
    expect(tab.text()).toContain(words.heightAt('minutes', 120))
    expect(tab.findAll('.control__bought').map((one) => one.text())).toContain('80 cards a sitting')
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
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(0)
    expect(tab.findAll('.control__number')).toHaveLength(0)
    expect(tab.get('.control__ends').text()).toBe('')
  })

  it('draws the picture and nothing waiting once the answer has landed', () => {
    const { tab } = drawn()
    expect(tab.findAll('.control__waiting')).toHaveLength(0)
    expect(tab.text()).not.toContain(words.waiting)
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(1)
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
  // says nothing. A run of nothing is that band, and its one number stands on
  // the foot the run lies along.
  it('says the one value once where the curve never moves', () => {
    const flat = drawn({ at: [point(), point(), point()] })
    const numbers = flat.tab.findAll('.control__number:not(.control__number--knob)')
    expect(numbers.map((one) => one.text())).toStrictEqual([words.heightAt('minutes', 0)])
  })

  // A count does not go below nothing, so nothing is the foot of the picture
  // under every goal and nothing is drawn under it.
  it('lays a curve of nothing along the foot, under every goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = drawn({ goal, at: [point(), point(), point(), point()] }, { goal })
      const drawnAt = heights(tab.get('.control__line').attributes('d') ?? '')
      expect(new Set(drawnAt).size).toBe(1)
      expect(drawnAt[0]).toBe(FOOT)
    }
  })

  // A run that touches nothing touches the foot, and one that never gets near
  // it is still read against a foot of nothing.
  it('stands the foot at nothing whether or not the run gets there', () => {
    const touching = drawn({
      at: [point(), point({ reviews: 40 }), point({ reviews: 80 })],
    })
    expect(heights(touching.tab.get('.control__line').attributes('d') ?? '')[0]).toBe(FOOT)

    const clear = drawn({
      at: [point({ reviews: 40 }), point({ reviews: 45 }), point({ reviews: 41 })],
    })
    const away = heights(clear.tab.get('.control__line').attributes('d') ?? '')
    expect(away.every((one) => one < FOOT)).toBe(true)
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
    const slider = tab.get('.control__picture[role="slider"]')
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
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'Home' })
    expect(ends()).toStrictEqual(['', words.widthAt('minutes', 30)])
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'End' })
    expect(ends()).toStrictEqual([words.widthAt('minutes', 0), ''])
  })

  it('carries the value at the knob, and it follows the knob', async () => {
    const { tab } = drawn()
    const at = () => tab.get('.control__number--knob')
    expect(at().text()).toBe(words.widthAt('minutes', 20))
    await tab.get('.control__picture[role="slider"]').trigger('keydown', { key: 'End' })
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
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(0)
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
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(1)
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
    expect(tab.findAll('.control__picture[role="slider"]')).toHaveLength(1)
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
      words.fieldName('load'),
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
      words.fieldName('load'),
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
      words.fieldName('load'),
      words.fieldName('evenLoad'),
    ])
    expect(rows(tab)).not.toContain(words.fieldName('backlog'))
    const day = tab.get('.preset__day')
    expect((day.element as HTMLInputElement).value).toBe('2026-09-29')
    // A day is chosen in one gesture, so choosing it is being done with it.
    await day.setValue('2026-10-09')
    expect(done).toStrictEqual(['types byDate 2026-10-09', 'settles'])
  })
})

describe('the settings under the control', () => {
  it('draws a row for each, with what it is beside it', () => {
    const { tab } = drawn()
    expect(tab.text()).toContain(words.fieldName('minutesADay'))
    expect(tab.text()).toContain(words.fieldDetail('minutesADay'))
    expect(tab.text()).toContain(words.fieldName('load'))
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
    expect(done).toStrictEqual(['types counts shows', 'settles'])
  })

  // The share of a day the debt takes is moved along its whole range, so any
  // of the hundred stands and none of them is a named position. The track
  // draws no figure, so the per cent is read out beside it.
  it('moves the share of a day the debt takes along a track, and reads it out', async () => {
    const { tab, done } = drawn({}, { backlog: 70 })
    expect(tab.text()).toContain(words.fieldName('backlog'))
    const row = tab
      .findAll('.preset__row')
      .find((one) => one.get('.preset__name').text() === words.fieldName('backlog'))

    await tab.vm.$nextTick()
    const track = row?.get('[role="slider"]')
    expect(track?.attributes('aria-valuenow')).toBe('70')
    expect(track?.attributes('aria-valuemin')).toBe('0')
    expect(track?.attributes('aria-valuemax')).toBe('100')
    expect(track?.attributes('aria-labelledby')).toBe('preset-backlog')
    expect(row?.get('.preset__percent').text()).toBe(words.percent(70))

    // The value is handed on, and the row is written once the handle rests.
    await track?.trigger('keydown', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['types backlog 69', 'settles'])
    // The hundred is the whole of it, and nothing outside it is taken.
    expect(BOUNDS.backlog).toStrictEqual({ least: 0, most: 100 })
  })

  // A row whose number the goal would fill in itself says so, and carries the
  // way back to what the goal produces. The mark is a picture, so it says in
  // words what it is for.
  it('marks a row standing at a value of its own, and offers it back to the goal', async () => {
    const { tab, done, byHand } = drawn({ goal: 'retention' }, { goal: 'retention' })
    byHand.value = new Set(['reviewsADay'])
    await tab.vm.$nextTick()

    const mine = tab.findAll('.preset__row--mine')
    expect(mine).toHaveLength(1)
    expect(mine[0]?.get('.preset__name').text()).toBe(words.fieldName('reviewsADay'))
    expect(mine[0]?.get('.preset__mine').text()).toBe(words.byHand)

    const back = mine[0]?.get('.preset__follows')
    expect(back?.attributes('aria-label')).toBe(words.follows)
    expect(back?.attributes('title')).toBe(words.follows)
    await back?.trigger('click')
    expect(done).toStrictEqual(['follows reviewsADay'])
  })

  it('leaves a row that follows the goal unmarked, with nothing offered back', () => {
    const { tab } = drawn({ goal: 'retention' }, { goal: 'retention' })
    expect(tab.findAll('.preset__row--mine')).toHaveLength(0)
    expect(tab.findAll('.preset__follows')).toHaveLength(0)
  })
})
