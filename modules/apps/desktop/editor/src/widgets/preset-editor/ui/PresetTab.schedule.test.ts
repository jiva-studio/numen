/**
 * What a preset schedules, and what it says when it schedules nothing.
 *
 * A goal draws the settings it schedules by and no others, and a preset with
 * nothing to work on says so where the picture would stand.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { StopReason } from '@numen/protocol'
import PresetTab from './PresetTab.vue'
import { NOWHERE, type Curve } from '../types'
import { drawn, point, rows, tabAt } from '../drawn'
import { WORDS as words } from '../words'

describe('a preset that schedules nothing', () => {
  /** Every verdict the schema carries, read off the schema itself. */
  const VERDICTS = Object.values(StopReason).filter(
    (one): one is StopReason => typeof one === 'number',
  )

  /** Those of them a person is told something about. */
  const STOPPING = VERDICTS.filter(
    (one) => one !== StopReason.NOTHING && one !== StopReason.UNSPECIFIED,
  )

  /** The tab drawn for a preset stopped for that reason. */
  const stopped = (why: StopReason) => {
    const { state } = tabAt()
    return mount(PresetTab, { props: { state: { ...state, stopped: ref(why) } } })
  }

  it('has a sentence of its own for each verdict there is', () => {
    const said = STOPPING.map((one) => words.stopped(one))
    expect(said.filter((one) => one !== '')).toHaveLength(STOPPING.length)
    expect(new Set(said).size).toBe(STOPPING.length)
  })

  it('says why, in the words that verdict has', () => {
    for (const why of STOPPING) {
      expect(stopped(why).get('[data-preset="stopped"]').text(), `${why}`).toBe(words.stopped(why))
    }
  })

  it('says nothing at all of a preset that schedules', () => {
    expect(stopped(StopReason.NOTHING).findAll('[data-preset="stopped"]')).toHaveLength(0)
    expect(stopped(StopReason.UNSPECIFIED).findAll('[data-preset="stopped"]')).toHaveLength(0)
  })

  // A goal of a date reading no day, and a day of the week carrying none of the
  // load: neither is a reason the tab could reach on its own.
  it('says the reasons only the vault knows', () => {
    expect(stopped(StopReason.NO_DAY).text()).toContain(words.stopped(StopReason.NO_DAY))
    expect(stopped(StopReason.NO_LOAD).text()).toContain(words.stopped(StopReason.NO_LOAD))
    expect(stopped(StopReason.NO_WEEK).text()).toContain(words.stopped(StopReason.NO_WEEK))
  })

  // One quiet day promises a next day that carries some load, and a week at
  // nothing has none to promise.
  it('promises a next day for one quiet day and not for a quiet week', () => {
    expect(words.stopped(StopReason.NO_LOAD)).toContain('next day')
    expect(words.stopped(StopReason.NO_WEEK)).not.toContain('next day')
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
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(0)
    expect(tab.text()).not.toContain('holds 0 cards')
  })

  it('says the decks pointing here hold no cards, where they do point here', () => {
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).toContain(words.noCards(1))
    expect(drawn({ ...nothing, decks: 4 }).tab.text()).toContain(words.noCards(4))
    expect(drawn({ ...nothing, decks: 1 }).tab.text()).not.toContain(words.unpointed)
  })

  // The preset schedules and there is nothing here it can schedule. It is said
  // in the control's place, and it is not a reason the preset is stopped.
  it('says a material nobody has begun that no place of the range begins', () => {
    const { tab } = drawn({ ...nothing, decks: 1, cards: 900, unbegun: 900 })
    expect(tab.text()).toContain(words.beginsNothing)
    expect(tab.findAll('[data-preset="stopped"]')).toHaveLength(0)
  })

  // A preset holding cards is never told it holds none, so a curve of zeros
  // keeps its control and says nothing about the vault.
  it('draws the control for a preset holding cards, whatever its curve comes to', () => {
    const { tab } = drawn({ ...nothing, decks: 4, cards: 900 })
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(1)
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
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(1)
  })

  it('leaves its settings there to be set up before a deck points here', () => {
    const { tab, done } = drawn(nothing)
    expect(tab.findAll('[data-preset-row]')).toHaveLength(6)
    const minutes = tab
      .findAll('[data-preset-row]')
      .find((one) => one.get('[data-preset="name"]').text() === words.fieldName('minutesADay'))
    minutes?.get('input').setValue('7')
    expect(done).toStrictEqual(['types minutesADay 7'])
  })
})

describe('the settings the chosen goal schedules by', () => {
  /** What each row of the receipt is called, which is what the goal draws. */
  // A goal names one budget. The budgets of the other two are not drawn, so
  // nothing on the screen offers to close a day by a measure nobody named.
  it('draws the minutes alone under a goal of minutes', () => {
    const { tab } = drawn()
    expect(rows(tab)).toStrictEqual([
      words.fieldName('learned'),
      words.fieldName('interval'),
      words.fieldName('minutesADay'),
      words.fieldName('backlog'),
      words.fieldName('load'),
      words.fieldName('evenLoad'),
    ])
    expect(tab.findAll('[data-preset="day"]')).toHaveLength(0)
  })

  it('draws the target and the counts that close a day under a goal of retention', () => {
    const { tab } = drawn({ goal: 'retention' }, { goal: 'retention' })
    expect(rows(tab)).toStrictEqual([
      words.fieldName('newADay'),
      words.fieldName('reviewsADay'),
      words.fieldName('learned'),
      words.fieldName('interval'),
      words.fieldName('retention'),
      words.fieldName('counts'),
      words.fieldName('backlog'),
      words.fieldName('load'),
      words.fieldName('evenLoad'),
    ])
    expect(tab.findAll('[data-preset="day"]')).toHaveLength(0)
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
      words.fieldName('learned'),
      words.fieldName('interval'),
      words.fieldName('byDate'),
      words.fieldName('load'),
      words.fieldName('evenLoad'),
    ])
    expect(rows(tab)).not.toContain(words.fieldName('backlog'))
    const day = tab.get('[data-preset="day"]')
    expect((day.element as HTMLInputElement).value).toBe('2026-09-29')
    // A day is chosen in one gesture, so choosing it is being done with it.
    await day.setValue('2026-10-09')
    expect(done).toStrictEqual(['types byDate 2026-10-09', 'settles'])
  })
})

