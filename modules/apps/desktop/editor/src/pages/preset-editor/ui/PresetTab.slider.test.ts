/**
 * The picture the one slider is, drawn in a document.
 *
 * The curve is the track: one stop on the way round the screen, walked by the
 * arrow keys, and read back off wherever a pointer stands on it. What is asked
 * here is what the picture draws and what a gesture on it comes to.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PresetTab from './PresetTab.vue'
import { NOWHERE, type Point } from '../types'
import { clearBacklog } from '../lib/curve'
import { FOOT } from '../lib/plot'
import { mountPresetTab, heights, point, tabAt } from '../fixtures'
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

  it('walks the grid a place at a time, and writes once the key is let go of', async () => {
    const { tab, done } = mountPresetTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keyup', { key: 'ArrowRight' })
    expect(done).toStrictEqual(['move 3', 'settle'])
  })

  it('walks to either end, and no further', async () => {
    const { tab, done } = mountPresetTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    await control.trigger('keydown', { key: 'End' })
    await control.trigger('keydown', { key: 'ArrowRight' })
    await control.trigger('keydown', { key: 'Home' })
    await control.trigger('keydown', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['move 3', 'move 3', 'move 0', 'move 0'])
  })

  it('leaves a keystroke that is nobody’s to the window', async () => {
    const { tab, done } = mountPresetTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'k' })
    expect(done).toStrictEqual([])
  })

  it('offers the three goals by the value each steers, under a label saying so', () => {
    const { tab } = mountPresetTab()
    expect(tab.get('[data-preset="label"]').text()).toBe(words.goal)
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(tab.text()).toContain(words.goalName(goal))
    }
  })

  // A name inside the picture is scaled with it and is set at no step of the
  // page's type.
  // The knob is where the person put it and needs no telling. The other mark
  // is not obvious and carries its name.
  it('names the mark that needs a name, and leaves the knob unnamed', () => {
    const { tab } = mountPresetTab()
    const names = tab.findAll('[data-control="label"]').map((one) => one.text())
    expect(names).toStrictEqual([words.markName('minutes')])
    expect(tab.text()).not.toContain('you are here')
    expect(tab.get('svg').find('text').exists()).toBe(false)
  })

  // A person moving the control needs to know what it is acting on, which the
  // picture itself never says.
  // A readout of what is being steered, scanned and not read: each figure its
  // own tile, with the word for what it counts under it.
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
  it('leaves out a tile whose figure stands at nothing', () => {
    const { tab } = mountPresetTab({ decks: 4, cards: 160, overdue: 0, unbegun: 0 })
    expect(tab.findAll('[data-control="material"] [data-control="tile"]')).toHaveLength(2)
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
    const { tab } = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 12, 30] }), point()],
    })
    const rules = tab.findAll('[data-control="rule"]')
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
    const { tab } = mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog: [9, 4, 0, 0] }), point()],
    })
    const said = tab
      .get('[data-control="callout"]')
      .findAll('[data-control="bought"]')
      .map((one) => one.text())
    expect(said).toStrictEqual(['20 minutes a day', '80 cards a session', 'overdue gone in 3 days'])
    // Its own value on the width stays on its own line under the picture.
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.widthAt('minutes', 20),
    )
    // The tail is on the knob, so the numbers belong to that place and not to
    // the picture as a whole.
    expect(tab.findAll('[data-control="tail"]')).toHaveLength(1)
  })

  it('says what a target costs, and a date the days and the day’s length', () => {
    const kept = mountPresetTab(
      {
        goal: 'retention',
        grid: [0.7, 0.8, 0.9, 0.95],
        now: { at: 2, value: 0.9, day: '' },
        at: [point(), point(), point({ minutes: 14, backlog: [5, 0] }), point()],
      },
      { goal: 'retention' },
    )
    expect(kept.tab.findAll('[data-control="bought"]').map((one) => one.text())).toStrictEqual([
      '90% remembered',
      '14 minutes a day',
      'overdue gone in 2 days',
    ])

    // Under a date every card is to be got through anyway, so when the overdue
    // goes says nothing and no line is drawn for it.
    const dated = mountPresetTab(
      {
        goal: 'date',
        grid: [10, 20, 30, 40],
        days: ['2026-09-09', '2026-09-19', '2026-09-29', '2026-10-09'],
        now: { at: 2, value: 30, day: '2026-09-29' },
        at: [point(), point(), point({ minutes: 40, backlog: [9, 4, 0, 8] }), point()],
      },
      { goal: 'date', byDate: '2026-09-29' },
    )
    expect(dated.tab.findAll('[data-control="bought"]').map((one) => one.text())).toStrictEqual([
      '30 days off',
      '40 minutes a day',
    ])
  })

  // Nothing overdue is nothing to say about it, and a pile the days projected
  // never clear is said in words and not as a figure nobody can stand behind.
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
    const { tab } = mountPresetTab({ grid: [0, 10, 20, 30], now: { at: 2, value: 23, day: '' } })
    const first = () => tab.findAll('[data-control="bought"]')[0]?.text()
    expect(first()).toBe('23 minutes a day')
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.widthAt('minutes', 23),
    )
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(first()).toBe('30 minutes a day')
    expect(tab.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.widthAt('minutes', 30),
    )
  })

  // The figure is read off the very run the backlog is drawn from, so the picture
  // and the words can never disagree.
  it('takes the day it goes off the projection the backlog is drawn from', () => {
    expect(clearBacklog([9, 4, 0, 0])).toBe(3)
    expect(clearBacklog([9, 21, 40])).toBe(-1)
    expect(clearBacklog([0, 0, 0])).toBe(null)
    expect(clearBacklog([])).toBe(null)
  })

  // A bubble that covered the line would hide the thing it is about.
  it('stands over the knob, and under it where over would leave the picture', async () => {
    const { tab } = mountPresetTab()
    // Which way the bubble is lifted off its anchor, which is the anchor the
    // tail sits on.
    const lift = () =>
      /translate:[^;]*\s([^\s;]+);?/.exec(
        tab.get('[data-control="callout"]').attributes('style') ?? '',
      )?.[1]
    const getTailUnder = () => tab.get('[data-control="tail"]').attributes('data-under')
    const slider = tab.get('[data-control="picture"][role="slider"]')

    // At the foot of this curve there is room above the knob.
    await slider.trigger('keydown', { key: 'Home' })
    expect(lift()).toBe('-100%')
    expect(getTailUnder()).toBeUndefined()

    // At its top there is none, and the bubble and its tail turn over together.
    await slider.trigger('keydown', { key: 'End' })
    expect(lift()).toBe('0')
    expect(getTailUnder()).toBeDefined()
  })

  // The picture is what the keyboard comes to and what it moves. What focus is
  // drawn as on it is a browser's answer, and is asked in the stories.
  it('is what the keyboard reaches, and it takes the focus', () => {
    const one = tabAt()
    const tab = mount(PresetTab, { props: { state: one.state }, attachTo: document.body })
    const picture = tab.get('[data-control="picture"][role="slider"]')

    expect(picture.attributes('tabindex')).toBe('0')
    ;(picture.element as HTMLElement).focus()
    expect(document.activeElement).toBe(picture.element)

    tab.unmount()
  })

  // The drop line and the value under it belong to the knob, so both stand
  // where the person put it and nowhere else.
  it('drops its line from the knob, wherever the knob is', async () => {
    const { tab } = mountPresetTab()
    const dropAt = () => tab.get('[data-control="drop"]').attributes('x1')
    const knobAt = () => tab.get('[data-control="knob"]').attributes('cx')
    expect(dropAt()).toBe(knobAt())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(dropAt()).toBe(knobAt())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(dropAt()).toBe(knobAt())
  })

  // The mark keeps its short name on the picture, and the paragraph says the
  // value it stands at and what that figure means. Nothing is captioned.
  it('draws and says nothing where the curve answers no suggestion', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal, suggested: NOWHERE }, { goal })
      expect(tab.findAll('[data-control="suggested"]')).toHaveLength(0)
      expect(tab.findAll('[data-control="label"]')).toHaveLength(0)
      expect(tab.text()).not.toContain(words.markName(goal))
    }
    // The knob and its number are the person's own and stand either way.
    expect(
      mountPresetTab({ suggested: NOWHERE }).tab.findAll('[data-control="knob"]'),
    ).toHaveLength(1)
    expect(
      mountPresetTab({ suggested: NOWHERE }).tab.findAll('[data-control="number"][data-at-knob]'),
    ).toHaveLength(1)
  })
})

// A day can leave a card less time than the rule wants, and no pace reaches
// it. That is arithmetic, so it is a count said beside what the place buys,
// where a person choosing the day is already reading.
describe('what a day cannot reach', () => {
  const getBoughtTexts = (tab: ReturnType<typeof mount>) =>
    tab.findAll('[data-control="bought"]').map((one) => one.text())

  const createDatedTab = (over: Partial<Point> = {}) =>
    mountPresetTab(
      {
        goal: 'date',
        grid: [14, 30, 90, 180],
        days: ['2026-09-14', '2026-09-30', '2026-11-29', '2027-02-27'],
        cards: 79,
        now: { at: 0, value: 14, day: '2026-09-14' },
        at: [
          point({ minutes: 40, short: 38, ...over }),
          point({ minutes: 30, short: 11 }),
          point({ minutes: 20, short: 0 }),
          point({ minutes: 15, short: 0 }),
        ],
      },
      { goal: 'date', byDate: '2026-09-14' },
    )

  it('says how many no pace reaches, beside what the day costs', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(getBoughtTexts(tab)).toStrictEqual([
      '14 days off',
      '40 minutes a day',
      '38 of 79 cannot get there',
    ])
  })

  it('says nothing of it where the day leaves every card time enough', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    const said = getBoughtTexts(tab)
    expect(said).toStrictEqual(['180 days off', '15 minutes a day'])
    expect(said.join(' ')).not.toContain('cannot get there')
    expect(said.join(' ')).not.toContain('0 of')
  })

  it('follows the day, since each day leaves its own cards short', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    await tab
      .get('[data-control="picture"][role="slider"]')
      .trigger('keydown', { key: 'ArrowRight' })
    expect(getBoughtTexts(tab)).toContain('11 of 79 cannot get there')
  })

  // It is the goal of a date that names a day, so it is that goal alone that
  // has a day nothing reaches by.
  it('says it under a goal of a date and under neither of the others', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ reviews: 80, short: 38 }), point()],
    })
    expect(getBoughtTexts(tab).join(' ')).not.toContain('cannot get there')
  })
})

describe('what the control stands at', () => {
  it('is read out in the units of its goal, with what it buys over the knob', () => {
    const { tab } = mountPresetTab()
    expect(tab.text()).toContain(words.value('minutes', 20, ''))
    expect(tab.findAll('[data-control="bought"]').map((one) => one.text())).toContain(
      '80 cards a session',
    )
  })

  // Nobody reads under the picture while dragging, so what a place buys is
  // said in the bubble and the room under the picture is the axis alone.
  it('leaves the axis and what belongs to it under the picture', () => {
    const { tab } = mountPresetTab()
    const foot = tab.findAll('[data-control="foot"]')[0]
    expect(foot?.get('[data-control="number"][data-at-knob]').text()).toBe(
      words.widthAt('minutes', 20),
    )
    expect(foot?.get('[data-control="name"][data-axis="x"]').text()).toBe(words.axisX('minutes'))
    // Nothing under the picture but the axis: what a place buys is said in the
    // bubble over the knob, and nowhere else.
    expect(foot?.text()).not.toContain(words.value('minutes', 20, ''))
  })

  // A point read as zero draws a screen of zeroes, which reads as a broken one.
  it('reads the cards off the place the knob stands at', () => {
    const { tab } = mountPresetTab()
    const said = tab.findAll('[data-control="bought"]').map((one) => one.text())
    expect(said).toContain('80 cards a session')
    expect(said).not.toContain('0 cards a session')
  })

  // The figure is what a person would recall when a card comes round, said as
  // a percentage and named as an act of remembering.
  it('speaks the share as a percentage, and of remembering', () => {
    expect(words.value('retention', 0.9, '')).toBe('90% remembered')
    expect(words.widthAt('retention', 0.9)).toBe('90%')
    expect(words.fieldDetail('retention')).toContain('remember')
    for (const said of [
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
      expect(said).not.toContain('0.9')
    }
  })

  // The height, the bubble over the knob and the axis are one number: every
  // card the session puts in front of the person, new and returning.
  it('names the height, the bubble and the axis in cards of a session', () => {
    const { tab } = mountPresetTab()
    expect(tab.get('[data-control="name"][data-axis="y"]').text()).toBe(words.axisY('minutes'))
    expect(tab.get('[data-control="name"][data-axis="y"]').text()).toContain('session')
    expect(tab.text()).toContain(words.heightAt('minutes', 120))
    expect(tab.findAll('[data-control="bought"]').map((one) => one.text())).toContain(
      '80 cards a session',
    )
  })

  // The window's own arithmetic never reaches the eye as a figure now: until
  // the first answer lands the picture says it is reading the vault, and no
  // tile, no axis number and no readout is drawn.
  it('draws no figure at all until the answer lands', () => {
    const { tab } = mountPresetTab({ honest: false })
    expect(tab.get('[data-control="waiting"]').text()).toContain(words.waiting)
    expect(tab.findAll('[data-control="tile"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="number"]')).toHaveLength(0)
    expect(tab.get('[data-control="ends"]').text()).toBe('')
  })

  // What the tiles count is a fact about the material, and no setting moves
  // one of those figures. They stand at what the window was last told while
  // the curve of the settings a person is moving is worked out.
  it('keeps the figures the material stands at while a curve is on its way', () => {
    const { tab } = mountPresetTab({ honest: false }, {}, true, {
      decks: 4,
      cards: 160,
      overdue: 45,
      unbegun: 30,
    })
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
    const { tab } = mountPresetTab({ honest: false })
    expect(tab.findAll('[data-control="waiting"]')).toHaveLength(1)
    expect(tab.text()).toContain(words.waiting)
    expect(tab.findAll('[data-control="picture"], [data-backlog="picture"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="number"]')).toHaveLength(0)
    expect(tab.get('[data-control="ends"]').text()).toBe('')
  })

  // A caption promising work in progress is a promise, and there is nothing
  // behind it once the vault has refused the picture.
  it('says nothing of reading the vault where no answer is coming', () => {
    const { tab } = mountPresetTab({ honest: false }, {}, false)
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

  // The rows under the picture keep their room, so the answer landing moves
  // nothing below the plot.
  it('keeps the rows under the picture whether or not the answer has landed', () => {
    for (const one of [mountPresetTab({ honest: false }), mountPresetTab()]) {
      expect(one.tab.findAll('[data-control="under"]')).toHaveLength(1)
      // The picture's own row of ends, and the backlog's under it.
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
    expect(words.axisX('minutes')).toBe('Minutes a day')
    expect(words.axisY('retention')).toBe('Minutes a day')
    expect(words.axisX('retention')).toBe('Retention')
    expect(words.axisY('date')).toBe('Minutes a day')
    expect(words.axisX('date')).toBe('Days from today')
  })

  // The words say which way is better and the numbers say how much, so a
  // height can be read off the picture and a place along it can be told.
  it('carries the ends of the backlog, against the lines they are the height of', () => {
    const numbers = mountPresetTab()
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    // The extent of the fixture runs from no cards a day to a hundred and twenty.
    expect(numbers).toContain(words.heightAt('minutes', 120))
  })

  // A number the line or a mark stands on is dropped: the axis gives way, and
  // the drawing keeps what it has to say.
  it('drops an axis number the drawing stands on rather than print over it', () => {
    // The fixture's curve leaves the foot at the left edge, where the low
    // number would be set.
    const numbers = mountPresetTab()
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    expect(numbers).not.toContain(words.heightAt('minutes', 0))
  })

  // Two ends of an extent of no width are one number, and one number said twice
  // says nothing. A run of nothing is that extent, and its one number stands on
  // the foot the run lies along.
  it('says the one value once where the curve never moves', () => {
    const flat = mountPresetTab({ at: [point(), point(), point()] })
    const numbers = flat.tab.findAll('[data-control="number"]:not([data-at-knob])')
    expect(numbers.map((one) => one.text())).toStrictEqual([words.heightAt('minutes', 0)])
  })

  // A count does not go below nothing, so nothing is the foot of the picture
  // under every goal and nothing is drawn under it.
  it('lays a curve of nothing along the foot, under every goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal, at: [point(), point(), point(), point()] }, { goal })
      const drawnAt = heights(tab.get('[data-control="line"]').attributes('d') ?? '')
      expect(new Set(drawnAt).size).toBe(1)
      expect(drawnAt[0]).toBe(FOOT)
    }
  })

  // A run that touches nothing touches the foot, and one that never gets near
  // it is still read against a foot of nothing.
  it('stands the foot at nothing whether or not the run gets there', () => {
    const touching = mountPresetTab({
      at: [point(), point({ reviews: 40 }), point({ reviews: 80 })],
    })
    expect(heights(touching.tab.get('[data-control="line"]').attributes('d') ?? '')[0]).toBe(FOOT)

    const clear = mountPresetTab({
      at: [point({ reviews: 40 }), point({ reviews: 45 }), point({ reviews: 41 })],
    })
    const away = heights(clear.tab.get('[data-control="line"]').attributes('d') ?? '')
    expect(away.every((one) => one < FOOT)).toBe(true)
  })

  // The knob's value rides a line of its own, so a knob at either end cannot
  // print over a number read off the picture.
  it('keeps the knob’s value on its own line, clear of the picture’s numbers', async () => {
    const { tab } = mountPresetTab()
    const over = tab.get('[data-control="over"]')
    const under = tab.get('[data-control="under"]')
    expect(under.findAll('[data-control="number"][data-at-knob]')).toHaveLength(1)
    expect(over.findAll('[data-control="number"][data-at-knob]')).toHaveLength(0)

    // At either end the knob's value is pulled back inside the picture's width.
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

  it('carries the value at either end of the range, and no words beside them', () => {
    const ends = mountPresetTab().tab.get('[data-control="ends"]').text()
    expect(ends).toContain(words.widthAt('minutes', 0))
    expect(ends).toContain(words.widthAt('minutes', 30))
    expect(ends).not.toContain('·')
  })

  // The knob says the value it stands on, and the end under it says nothing.
  it('leaves the end the knob stands on to the knob', async () => {
    const { tab } = mountPresetTab()
    const endTexts = () =>
      tab
        .findAll('[data-control="ends"]')[0]
        ?.findAll('span')
        .map((one) => one.text())
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(endTexts()).toStrictEqual(['', words.widthAt('minutes', 30)])
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(endTexts()).toStrictEqual([words.widthAt('minutes', 0), ''])
  })

  it('carries the value at the knob, and it follows the knob', async () => {
    const { tab } = mountPresetTab()
    const getKnobNumber = () => tab.get('[data-control="number"][data-at-knob]')
    expect(getKnobNumber().text()).toBe(words.widthAt('minutes', 20))
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(getKnobNumber().text()).toBe(words.widthAt('minutes', 30))
  })

  it('reads the numbers of each goal in that goal’s own units', () => {
    expect(words.widthAt('retention', 0.8)).toBe('80%')
    expect(words.widthAt('date', 12)).toBe('12 d')
    expect(words.heightAt('date', 45)).toBe('45 min')
    expect(words.heightAt('minutes', 80)).toBe('80 cards')
  })
})

// The verdict is the vault's, so the tab draws what it was handed and works
// nothing out of the settings in front of it.
