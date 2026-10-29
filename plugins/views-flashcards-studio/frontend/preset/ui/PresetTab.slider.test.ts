/**
 * The curve plot as the single slider control.
 *
 * The curve is the slider track: one tab stop, moved by the arrow keys, with
 * pointer positions mapped to grid values. The tests cover what the plot renders
 * and what a gesture on it evaluates to.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PresetTab from './PresetTab.vue'
import { NO_CURVE_POINT, type Point } from '../types'
import { clearBacklog } from '../lib/curve'
import { PLOT_BOTTOM } from '../lib/plot'
import { mountPresetTab, heights, point, createTab } from '../fixtures'
import { WORDS as words } from '../words'

describe('the one slider', () => {
  it('is the curve itself, and one stop on the way round the screen', () => {
    const { tab } = mountPresetTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    expect(control.attributes('tabindex')).toBe('0')
    expect(control.attributes('aria-valuemin')).toBe('0')
    expect(control.attributes('aria-valuemax')).toBe('30')
    expect(control.attributes('aria-valuenow')).toBe('20')
    expect(control.attributes('aria-valuetext')).toBe(words.value('minutes', 20, ''))
  })

  it('walks the grid a place at a time, and writes once the key is release', async () => {
    const { tab, recorded } = mountPresetTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keyup', { key: 'ArrowRight' })
    expect(recorded).toStrictEqual(['move 3', 'flushPending'])
  })

  it('walks to either end, and no further', async () => {
    const { tab, recorded } = mountPresetTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    await control.trigger('keydown', { key: 'End' })
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keydown', { key: 'Home' })
    await control.trigger('keydown', { key: 'ArrowLeft' })
    expect(recorded).toStrictEqual(['move 3', 'move 3', 'move 0', 'move 0'])
  })

  it('leaves a keystroke that is nobody’s to the window', async () => {
    const { tab, recorded } = mountPresetTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'k' })
    expect(recorded).toStrictEqual([])
  })

  it('offers the three goals by the value each steers, under a label saying so', () => {
    const { tab } = mountPresetTab()
    expect(tab.get('[data-preset="label"]').text()).toBe(words.goal)
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(tab.text()).toContain(words.goalName(goal))
    }
  })

  // Labels inside the plot scale with it and are not set in the page's type steps.
  // The knob needs no label; the suggested mark carries its name.
  it('names the mark that needs a name, and leaves the knob unnamed', () => {
    const { tab } = mountPresetTab()
    const names = tab.findAll('[data-control="label"]').map((one) => one.text())
    expect(names).toStrictEqual([words.markName('minutes')])
    expect(tab.text()).not.toContain('you are here')
    expect(tab.get('svg').find('text').exists()).toBe(false)
  })

  // The material readout shows what the control acts on, which the plot does not
  // state. Each figure is its own tile with its word below it.
  it('says what the control is acting on, as tiles over the picture', () => {
    const { tab } = mountPresetTab({ decks: 4, cards: 160, overdue: 45, unbegun: 30 })
    const tiles = tab
      .findAll('[data-control="material"] [data-control="tile"]')
      .map((one) => [
        one.get('[data-control="figure"]').text(),
        one.get('[data-control="word"]').text(),
      ])
    expect(tiles).toStrictEqual([
      ['4', 'decks'],
      ['160', 'cards'],
      ['45', 'overdue'],
      ['30', 'new'],
    ])
  })

  // The tiles take an equal share of the width, so a figure standing at
  // nothing leaves the rest to spread over it.
  it('leaves out a tile whose figure remains at nothing', () => {
    const { tab } = mountPresetTab({ decks: 4, cards: 160, overdue: 0, unbegun: 0 })
    expect(tab.findAll('[data-control="material"] [data-control="tile"]')).toHaveLength(2)
  })

  // calls it. Overdue keeps its own word, which is a distinction of its own.
  it('classifies unreviewed card as new', () => {
    const transcriptText = words
      .material({ decks: 4, cards: 160, overdue: 45, unbegun: 30 })
      .map((one) => one.name)
    expect(transcriptText).toContain('new')
    expect(transcriptText).not.toContain('unbegun')
    expect(transcriptText).toContain('overdue')
  })

  it('leaves out whichever figure remains at nothing', () => {
    const formatMaterials = (overdueCount: number, freshCount: number) =>
      words
        .material({ decks: 1, cards: 160, overdue: overdueCount, unbegun: freshCount })
        .map((one) => `${one.figure} ${one.name}`)
    expect(formatMaterials(0, 30)).toStrictEqual(['1 deck', '160 cards', '30 new'])
    expect(formatMaterials(45, 0)).toStrictEqual(['1 deck', '160 cards', '45 overdue'])
    expect(formatMaterials(0, 0)).toStrictEqual(['1 deck', '160 cards'])
  })

  // A figure beside an area measures nothing. Each picture carries the two
  // lines its numbers are read against.
  it('draws both axes on both pictures, in the rule the window separates with', () => {
    const { tab } = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 12, 30] }), point()],
    })
    const rules = tab.findAll('[data-control="rule"]')
    expect(rules).toHaveLength(4)
    // Each picture has one line up its left side and one along its foot.
    const upright = rules.filter((one) => one.attributes('x1') === one.attributes('x2'))
    const horizontalRules = rules.filter((one) => one.attributes('y1') === one.attributes('y2'))
    expect(upright).toHaveLength(2)
    expect(horizontalRules).toHaveLength(2)
  })

  // The knob's value is shown on the line below the plot only, so the bubble over
  // the knob states the value first, then what that value buys.
  it('says what this place buys, in a bubble over the knob, value first', () => {
    const { tab } = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 4, 0, 0] }), point()],
    })
    const response = tab
      .get('[data-control="callout"]')
      .findAll('[data-control="bought"]')
      .map((one) => one.text())
    expect(response).toStrictEqual([
      '20 minutes a day',
      '80 cards a session',
      'overdue gone in 3 days',
    ])
    // The knob's value stays on its own line below the plot.
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.formatGoalValue('minutes', 20),
    )
    // The tail is on the knob, so the numbers belong to the knob position and not
    // to the plot as a whole.
    expect(tab.findAll('[data-control="tail"]')).toHaveLength(1)
  })

  it('says what a target costs, and a date the days and the day’s length', () => {
    const tabWrapper = mountPresetTab(
      {
        goal: 'retention',
        grid: [0.7, 0.8, 0.9, 0.95],
        now: { index: 2, value: 0.9, day: '' },
        at: [point(), point(), point({ minutes: 14, backlog: [5, 0] }), point()],
      },
      { goal: 'retention' },
    )
    expect(
      tabWrapper.tab.findAll('[data-control="bought"]').map((one) => one.text()),
    ).toStrictEqual(['90% remembered', '14 minutes a day', 'overdue gone in 2 days'])

    // Under a date goal every card is reviewed anyway, so no overdue-clearing line
    // is drawn.
    const dated = mountPresetTab(
      {
        goal: 'date',
        grid: [10, 20, 30, 40],
        days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
        now: { index: 2, value: 30, day: '2026-09-29' },
        at: [point(), point(), point({ minutes: 40, backlog: [9, 4, 0, 8] }), point()],
      },
      { goal: 'date', byDate: '2026-09-29' },
    )
    expect(dated.tab.findAll('[data-control="bought"]').map((one) => one.text())).toStrictEqual([
      '30 days off',
      '40 minutes a day',
    ])
  })

  // A zero overdue count draws no line, and a backlog that does not clear within
  // the projected days is stated in words and not as a figure.
  it('draws the overdue line only where there is something overdue', () => {
    const none = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [0, 0, 0] }), point()],
    })
    expect(none.tab.findAll('[data-control="bought"]').map((one) => one.text())).toStrictEqual([
      '20 minutes a day',
      '80 cards a session',
    ])

    const never = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 21, 40] }), point()],
    })
    expect(never.tab.findAll('[data-control="bought"]').map((one) => one.text())).toStrictEqual([
      '20 minutes a day',
      '80 cards a session',
      'not within 3 days',
    ])
  })

  // A preset's own value need not sit on the grid, and the place it opens at
  // is only the nearest one. The knob does not claim a value it is not at.
  it('reads the preset’s own value where it opens, and the place once walked', async () => {
    const { tab } = mountPresetTab({ grid: [0, 10, 20, 30], now: { index: 2, value: 23, day: '' } })
    const first = () => tab.findAll('[data-control="bought"]')[0]?.text()
    expect(first()).toBe('23 minutes a day')
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.formatGoalValue('minutes', 23),
    )
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(first()).toBe('30 minutes a day')
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.formatGoalValue('minutes', 30),
    )
  })

  // The figure is derived from the same run the backlog is drawn from, so the plot
  // and the text agree.
  it('takes the day it goes off the projection the backlog is drawn from', () => {
    expect(clearBacklog([9, 4, 0, 0])).toBe(3)
    expect(clearBacklog([9, 21, 40])).toBe(-1)
    expect(clearBacklog([0, 0, 0])).toBe(null)
    expect(clearBacklog([])).toBe(null)
  })

  // A bubble that covered the line would hide the thing it is about.
  it('remains over the knob, and under it where over would leave the picture', async () => {
    const { tab } = mountPresetTab()
    // Which way the bubble is lifted off its anchor, which is the anchor the
    const lift = () =>
      /translate:[^;]*\s([^\s;]+);?/.exec(
        tab.get('[data-control="callout"]').attributes('style') ?? '',
      )?.[1]
    const getTailUnder = () => tab.get('[data-control="tail"]').attributes('data-under')
    const slider = tab.get('[data-control="picture"][role="slider"]')

    // At the foot of this curve the bubble has space above the knob.
    await slider.trigger('keydown', { key: 'Home' })
    expect(lift()).toBe('-100%')
    expect(getTailUnder()).toBeUndefined()

    // At its top there is none, and the bubble and its tail turn over together.
    await slider.trigger('keydown', { key: 'End' })
    expect(lift()).toBe('0')
    expect(getTailUnder()).toBeDefined()
  })

  // The plot is the keyboard target and the element the keys move. The focus
  // indicator rendering is browser-specific and is covered in the stories.
  it('is what the keyboard reaches, and it takes the focus', () => {
    const one = createTab()
    const tab = mount(PresetTab, { props: { state: one.state }, attachTo: document.body })
    const picture = tab.get('[data-control="picture"][role="slider"]')

    expect(picture.attributes('tabindex')).toBe('0')
    ;(picture.element as HTMLElement).focus()
    expect(document.activeElement).toBe(picture.element)

    tab.unmount()
  })

  // The drop line and the value below it follow the knob position.
  it('drops its line from the knob, wherever the knob is', async () => {
    const { tab } = mountPresetTab()
    const getDropX = () => tab.get('[data-control="drop"]').attributes('x1')
    const getKnobX = () => tab.get('[data-control="knob"]').attributes('cx')
    expect(getDropX()).toBe(getKnobX())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(getDropX()).toBe(getKnobX())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(getDropX()).toBe(getKnobX())
  })

  // The suggested mark keeps its short name on the plot, and the paragraph states
  // the suggested value.
  it('draws and says nothing where the curve answers no suggestion', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal, suggested: NO_CURVE_POINT }, { goal })
      expect(tab.findAll('[data-control="suggested"]')).toHaveLength(0)
      expect(tab.findAll('[data-control="label"]')).toHaveLength(0)
      expect(tab.text()).not.toContain(words.markName(goal))
    }
    // The knob and its number are present in either case.
    expect(
      mountPresetTab({ suggested: NO_CURVE_POINT }).tab.findAll('[data-control="knob"]'),
    ).toHaveLength(1)
    expect(
      mountPresetTab({ suggested: NO_CURVE_POINT }).tab.findAll(
        '[data-control="number"][data-at-knob]',
      ),
    ).toHaveLength(1)
  })
})

// A day can leave a card less time than the rule requires, so no pace meets the
// goal. This is arithmetic: the count is shown with what the position buys, where
// the user choosing the day reads it.
describe('what a day cannot reach', () => {
  const getBoughtTexts = (tab: ReturnType<typeof mount>) =>
    tab.findAll('[data-control="bought"]').map((one) => one.text())

  const createDatedTab = (overrides: Partial<Point> = {}) =>
    mountPresetTab(
      {
        goal: 'date',
        grid: [14, 30, 90, 180],
        days: ['2026-09-14', '2026-09-30', '2026-11-29', '2027-02-27'],
        cards: 79,
        now: { index: 0, value: 14, day: '2026-09-14' },
        at: [
          point({ minutes: 40, short: 38, ...overrides }),
          point({ minutes: 30, short: 11 }),
          point({ minutes: 20, short: 0 }),
          point({ minutes: 15, short: 0 }),
        ],
      },
      { goal: 'date', byDate: '2026-09-14' },
    )

  it('says how many no pace reaches, along with what the day costs', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(getBoughtTexts(tab)).toStrictEqual([
      '14 days off',
      '40 minutes a day',
      '38 of 79 not reachable',
    ])
  })

  it('says nothing of it where the day leaves every card time enough', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    const transcriptText = getBoughtTexts(tab)
    expect(transcriptText).toStrictEqual(['180 days off', '15 minutes a day'])
    expect(transcriptText.join(' ')).not.toContain('not reachable')
    expect(transcriptText.join(' ')).not.toContain('0 of')
  })

  it('follows the day, since each day leaves its own cards short', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    await tab
      .get('[data-control="picture"][role="slider"]')
      .trigger('keydown', { key: 'ArrowRight' })
    expect(getBoughtTexts(tab)).toContain('11 of 79 not reachable')
  })

  // Only the date goal names a day, so only that goal has a day that cannot be
  // reached.
  it('says it under a goal of a date and under neither of the others', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ reviews: 80, short: 38 }), point()],
    })
    expect(getBoughtTexts(tab).join(' ')).not.toContain('not reachable')
  })
})

describe('what the control remains at', () => {
  it('is read out in the units of its goal, with what it buys over the knob', () => {
    const { tab } = mountPresetTab()
    expect(tab.text()).toContain(words.value('minutes', 20, ''))
    expect(tab.findAll('[data-control="bought"]').map((one) => one.text())).toContain(
      '80 cards a session',
    )
  })

  // The bubble states what a position buys during a drag, so the area below the
  // plot holds the axis only.
  it('leaves the axis and what belongs to it under the picture', () => {
    const { tab } = mountPresetTab()
    const foot = tab.findAll('[data-control="foot"]')[0]
    expect(foot?.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.formatGoalValue('minutes', 20),
    )
    expect(foot?.get('[data-control="name"][data-axis="x"]').text()).toBe(words.axisX('minutes'))
    // Only the axis is below the plot; what a position buys is stated in the bubble
    // over the knob.
    expect(foot?.text()).not.toContain(words.value('minutes', 20, ''))
  })

  // A point read as zero draws a screen of zeroes, which reads as a broken one.
  it('reads the cards off the place the knob remains at', () => {
    const { tab } = mountPresetTab()
    const transcriptText = tab.findAll('[data-control="bought"]').map((one) => one.text())
    expect(transcriptText).toContain('80 cards a session')
    expect(transcriptText).not.toContain('0 cards a session')
  })

  // The figure is the probability of recall when a card is due, shown as a
  // percentage and labelled as remembered.
  it('speaks the share as a percentage, and of remembering', () => {
    expect(words.value('retention', 0.9, '')).toBe('90% remembered')
    expect(words.formatGoalValue('retention', 0.9)).toBe('90%')
    expect(words.fieldDetail('retention')).toContain('recall')
    for (const response of [
      words.value('retention', 0.9, ''),
      words
        .buys('retention', {
          value: 0.9,
          reviews: 48,
          minutes: 14,
          horizon: 0,
          clears: null,
          short: 0,
          cards: 160,
        })
        .join(' '),
      words.fieldDetail('retention'),
    ]) {
      expect(response).not.toContain('0.9')
    }
  })

  // The height, the bubble over the knob and the axis share one unit: the cards
  // the session presents, new and returning.
  it('names the height, the bubble and the axis in cards of a session', () => {
    const { tab } = mountPresetTab()
    expect(tab.get('[data-control="name"][data-axis="y"]').text()).toBe(words.axisY('minutes'))
    expect(tab.get('[data-control="name"][data-axis="y"]').text()).toContain('session')
    expect(tab.text()).toContain(words.formatGoalCost('minutes', 120))
    expect(tab.findAll('[data-control="bought"]').map((one) => one.text())).toContain(
      '80 cards a session',
    )
  })

  // Until the first response arrives the plot shows an indexing state, and no
  // tile, axis number or readout is drawn.
  it('draws no figure at all until the answer lands', () => {
    const { tab } = mountPresetTab({ isHonest: false })
    expect(tab.get('[data-control="waiting"]').text()).toContain(words.waiting)
    expect(tab.findAll('[data-control="tile"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="number"]')).toHaveLength(0)
    expect(tab.get('[data-control="ends"]').text()).toBe('')
  })

  // The tiles count facts about the material, which no setting changes. They keep
  // the last received values while the curve for the edited settings is computed.
  it('keeps the figures the material remains at while a curve is on its way', () => {
    const { tab } = mountPresetTab(
      { isHonest: false },
      {},
      {
        isWaiting: true,
        counts: {
          decks: 4,
          cards: 160,
          overdue: 45,
          unbegun: 30,
        },
      },
    )
    const tiles = tab
      .findAll('[data-control="material"] [data-control="tile"]')
      .map((one) => [
        one.get('[data-control="figure"]').text(),
        one.get('[data-control="word"]').text(),
      ])
    expect(tiles).toStrictEqual([
      ['4', 'decks'],
      ['160', 'cards'],
      ['45', 'overdue'],
      ['30', 'new'],
    ])
    expect(tab.get('[data-control="waiting"]').text()).toContain(words.waiting)
  })

  // A line drawn before the answer has to move when it lands, and a picture
  // that moves reads as a glitch. Nothing is drawn until there is an answer.
  it('draws no line and no control while the curve is being worked out', () => {
    const { tab } = mountPresetTab({ isHonest: false })
    expect(tab.findAll('[data-control="waiting"]')).toHaveLength(1)
    expect(tab.text()).toContain(words.waiting)
    expect(tab.findAll('[data-control="picture"], [data-backlog="picture"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="number"]')).toHaveLength(0)
    expect(tab.get('[data-control="ends"]').text()).toBe('')
  })

  // A caption promising work in progress is a promise, and there is nothing
  it('says nothing of indexing the vault where no answer is coming', () => {
    const { tab } = mountPresetTab({ isHonest: false }, {}, { isWaiting: false })
    expect(tab.findAll('[data-control="waiting"]')).toHaveLength(0)
    expect(tab.text()).not.toContain(words.waiting)
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(0)
  })

  it('draws the picture and nothing waiting once the answer has landed', () => {
    const { tab } = mountPresetTab()
    expect(tab.findAll('[data-control="waiting"]')).toHaveLength(0)
    expect(tab.text()).not.toContain(words.waiting)
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(1)
  })

  // The rows below the plot keep their height, so the arriving response does not
  // move content under the plot.
  it('keeps the rows under the picture whether or not the answer has landed', () => {
    for (const one of [mountPresetTab({ isHonest: false }), mountPresetTab()]) {
      expect(one.tab.findAll('[data-control="under"]')).toHaveLength(1)
      // The plot's own row of ends, and the backlog row below it.
      expect(one.tab.findAll('[data-control="ends"]')).toHaveLength(2)
    }
  })

  // Each axis is named along the axis it names, with its unit in the name.
  it('names both axes where each axis is, under every goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal }, { goal })
      expect(tab.get('[data-control="name"][data-axis="y"]').text()).toBe(words.axisY(goal))
      expect(tab.get('[data-control="name"][data-axis="x"]').text()).toBe(words.axisX(goal))
    }
    expect(words.axisY('minutes')).toBe('Cards in a session')
    expect(words.axisX('minutes')).toBe('Minutes per day')
    expect(words.axisY('retention')).toBe('Minutes per day')
    expect(words.axisX('retention')).toBe('Retention')
    expect(words.axisY('date')).toBe('Minutes per day')
    expect(words.axisX('date')).toBe('Days from today')
  })

  // The words state which direction is better and the numbers state how much, so
  // a height and a position along the axis can both be read.
  it('carries the ends of the backlog, against the lines they are the height of', () => {
    const numbers = mountPresetTab()
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    // The extent of the fixture runs from no cards a day to a hundred and twenty.
    expect(numbers).toContain(words.formatGoalCost('minutes', 120))
  })

  // A number the line or a mark relies on is dropped: the axis gives way, and
  // the drawing keeps what it has to say.
  it('drops an axis number the drawing relies on and not print over it', () => {
    // The fixture's curve leaves the foot at the left edge, where the low
    // number would be set.
    const numbers = mountPresetTab()
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    expect(numbers).not.toContain(words.formatGoalCost('minutes', 0))
  })

  // A zero-width extent has equal ends, so the value is shown once. A flat run of
  // zeros is such an extent, and its single number is positioned at the plot foot.
  it('says the one value once where the curve never moves', () => {
    const flat = mountPresetTab({ at: [point(), point(), point()] })
    const numbers = flat.tab.findAll('[data-control="number"]:not([data-at-knob])')
    expect(numbers.map((one) => one.text())).toStrictEqual([words.formatGoalCost('minutes', 0)])
  })

  // A count is never negative, so zero is the plot foot under every goal and the
  // zero line is drawn there.
  it('lays a curve of nothing along the foot, under every goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal, at: [point(), point(), point(), point()] }, { goal })
      const drawnAt = heights(tab.get('[data-control="line"]').attributes('d') ?? '')
      expect(new Set(drawnAt).size).toBe(1)
      expect(drawnAt[0]).toBe(PLOT_BOTTOM)
    }
  })

  // A run that touches nothing touches the foot, and one that never gets near
  // it is still read against a foot of nothing.
  it('remains the foot at nothing whether or not the run gets there', () => {
    const touching = mountPresetTab({
      at: [point(), point({ reviews: 40 }), point({ reviews: 80 })],
    })
    expect(heights(touching.tab.get('[data-control="line"]').attributes('d') ?? '')[0]).toBe(
      PLOT_BOTTOM,
    )

    const clear = mountPresetTab({
      at: [point({ reviews: 40 }), point({ reviews: 45 }), point({ reviews: 41 })],
    })
    const away = heights(clear.tab.get('[data-control="line"]').attributes('d') ?? '')
    expect(away.every((one) => one < PLOT_BOTTOM)).toBe(true)
  })

  // The knob's value is on its own line, so a knob at either end does not overlap
  // an axis number.
  it('keeps the knob’s value on its own line, clear of the picture’s numbers', async () => {
    const { tab } = mountPresetTab()
    const overControl = tab.get('[data-control="over"]')
    const underControl = tab.get('[data-control="under"]')
    expect(underControl.findAll('[data-control="number"][data-at-knob]')).toHaveLength(1)
    expect(overControl.findAll('[data-control="number"][data-at-knob]')).toHaveLength(0)

    // At either end the knob's value is clamped inside the plot width.
    const slider = tab.get('[data-control="picture"][role="slider"]')
    await slider.trigger('keydown', { key: 'Home' })
    expect(tab.get('[data-control="number"][data-at-knob]').attributes('style')).toContain(
      'translate: 0 0',
    )
    await slider.trigger('keydown', { key: 'End' })
    expect(tab.get('[data-control="number"][data-at-knob]').attributes('style')).toContain(
      'translate: -100% 0',
    )
  })

  it('carries the value at either end of the range, and no words alongside them', () => {
    const ends = mountPresetTab().tab.get('[data-control="ends"]').text()
    expect(ends).toContain(words.formatGoalValue('minutes', 0))
    expect(ends).toContain(words.formatGoalValue('minutes', 30))
    expect(ends).not.toContain('·')
  })

  // The knob states its own value, and the axis end under the knob is omitted.
  it('leaves the end the knob relies on to the knob', async () => {
    const { tab } = mountPresetTab()
    const endTexts = () =>
      tab
        .findAll('[data-control="ends"]')[0]
        ?.findAll('span')
        .map((one) => one.text())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(endTexts()).toStrictEqual(['', words.formatGoalValue('minutes', 30)])
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(endTexts()).toStrictEqual([words.formatGoalValue('minutes', 0), ''])
  })

  it('carries the value at the knob, and it follows the knob', async () => {
    const { tab } = mountPresetTab()
    const getKnobNumber = () => tab.get('[data-control="number"][data-at-knob]')
    expect(getKnobNumber().text()).toBe(words.formatGoalValue('minutes', 20))
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(getKnobNumber().text()).toBe(words.formatGoalValue('minutes', 30))
  })

  it('reads the numbers of each goal in that goal’s own units', () => {
    expect(words.formatGoalValue('retention', 0.8)).toBe('80%')
    expect(words.formatGoalValue('date', 12)).toBe('12 d')
    expect(words.formatGoalCost('date', 45)).toBe('45 min')
    expect(words.formatGoalCost('minutes', 80)).toBe('80 cards')
  })
})

// The verdict is the vault's, so the tab draws what it was handed and works
// nothing out of the settings in front of it.
