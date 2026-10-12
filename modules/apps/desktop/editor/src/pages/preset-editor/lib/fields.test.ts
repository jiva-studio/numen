/**
 * Which settings a goal schedules by, and the one word that says their shape.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, type Settings } from '../types'
import { FIELDS, getGoalFields, getShapeKey, getGoalField } from './fields'

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

describe('the settings a goal schedules by', () => {
  // A goal names one budget, and the budgets of the other two take no part in
  // it: they are not drawn, so nothing on the screen offers to cut the day
  // short by a measure the person did not name.
  it('is the minutes alone under a goal of minutes', () => {
    const under = getGoalFields('minutes', DEFAULTS.learned)
    expect(under).toContain('minutesADay')
    expect(under).not.toContain('newADay')
    expect(under).not.toContain('reviewsADay')
    expect(under).not.toContain('retention')
    expect(under).not.toContain('byDate')
  })

  it('is the target and the counts that close a day under a goal of retention', () => {
    const under = getGoalFields('retention', DEFAULTS.learned)
    expect(under).toContain('retention')
    expect(under).toContain('newADay')
    expect(under).toContain('reviewsADay')
    expect(under).not.toContain('minutesADay')
    expect(under).not.toContain('byDate')
  })

  it('is the day alone under a goal of a date, which nothing else may cut short', () => {
    const under = getGoalFields('date', DEFAULTS.learned)
    expect(under).toContain('byDate')
    expect(under).not.toContain('minutesADay')
    expect(under).not.toContain('newADay')
    expect(under).not.toContain('reviewsADay')
  })

  it('draws what stands under no goal in particular under all three', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(getGoalFields(goal, DEFAULTS.learned)).toContain('load')
      expect(getGoalFields(goal, DEFAULTS.learned)).toContain('evenLoad')
    }
  })

  // What a day's budget is spent on is a measure in cards, so it stands under
  // the goal whose budget is cards and takes no part in the other two.
  it('draws what a budget is spent on under the goal whose budget is cards', () => {
    expect(getGoalFields('retention', DEFAULTS.learned)).toContain('counts')
    expect(getGoalFields('minutes', DEFAULTS.learned)).not.toContain('counts')
    expect(getGoalFields('date', DEFAULTS.learned)).not.toContain('counts')
  })

  // The share of a day that goes to the debt says what a day is spent on and
  // closes nothing. A goal of a date carries the whole material by its own
  // reckoning and has no part in it.
  it('draws the backlog share under the two goals that keep a budget of a day', () => {
    expect(getGoalFields('minutes', DEFAULTS.learned)).toContain('backlog')
    expect(getGoalFields('retention', DEFAULTS.learned)).toContain('backlog')
    expect(getGoalFields('date', DEFAULTS.learned)).not.toContain('backlog')
  })

  it('draws them in the order the receipt keeps them in', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const under = getGoalFields(goal, DEFAULTS.learned)
      const places = under.map((field) => FIELDS.indexOf(field))
      expect(places).toStrictEqual([...places].sort((a, b) => a - b))
    }
  })
})

// The rule for what is learned is drawn under every goal, and under it the one
// value that rule reads. The other keeps its value and takes no part.
describe('the rule for what is learned', () => {
  it('draws the rule under every goal, and the value the rule reads', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(getGoalFields(goal, 'interval')).toContain('learned')
      expect(getGoalFields(goal, 'interval')).toContain('interval')
      expect(getGoalFields(goal, 'retention')).toContain('learned')
      expect(getGoalFields(goal, 'retention')).not.toContain('interval')
    }
  })

  // One key is one row, whether it is read by the goal, by the rule, or by both.
  it('draws the target once, and only where something reads it', () => {
    const both = getGoalFields('retention', 'retention')
    expect(both.filter((one) => one === 'retention')).toHaveLength(1)
    expect(getGoalFields('minutes', 'retention')).toContain('retention')
    expect(getGoalFields('minutes', 'interval')).not.toContain('retention')
  })

  it('stands the rule over the value it reads', () => {
    for (const learned of ['interval', 'retention'] as const) {
      const under = getGoalFields('minutes', learned)
      const value = learned === 'interval' ? 'interval' : 'retention'
      expect(under.indexOf(value)).toBe(under.indexOf('learned') + 1)
    }
  })

  // A curve is asked again where a setting it is drawn from moves, and both of
  // these move it: what is learned is counted off the run.
  it('gives the curve its shape, both the rule and the days it reads', () => {
    const one = settings({ learned: 'interval', interval: 21 })
    expect(getShapeKey(one)).not.toBe(getShapeKey({ ...one, interval: 30 }))
    expect(getShapeKey(one)).not.toBe(getShapeKey({ ...one, learned: 'retention' }))
  })
})

describe('the one word that says the shape of the settings', () => {
  const every = settings({
    byDate: '2026-09-30',
    load: { tue: 50, mon: 25 },
  })

  it('names the goal, then every setting the goal steers by its value', () => {
    expect(getShapeKey({ ...every, goal: 'date' })).toBe(
      'date newADay=10 reviewsADay=200 learned=interval interval=21 retention=0.9 ' +
        'minutesADay=20 byDate=2026-09-30 counts=cards backlog=100 load=mon=25,tue=50 evenLoad=true',
    )
  })

  it('leaves the day out under a goal that does not getGoalField by it', () => {
    expect(getShapeKey({ ...every, goal: 'minutes' })).toBe(
      'minutes newADay=10 reviewsADay=200 learned=interval interval=21 retention=0.9 ' +
        'minutesADay=20 counts=cards backlog=100 load=mon=25,tue=50 evenLoad=true',
    )
  })

  it('differs where the reviews of a day, the backlog or the even load differ', () => {
    const shape = getShapeKey(every)

    expect(getShapeKey({ ...every, reviewsADay: 300 })).not.toBe(shape)
    expect(getShapeKey({ ...every, backlog: 50 })).not.toBe(shape)
    expect(getShapeKey({ ...every, evenLoad: false })).not.toBe(shape)
  })

  it('is the same wherever the days of the load were written first', () => {
    expect(getShapeKey({ ...every, load: { mon: 25, tue: 50 } })).toBe(getShapeKey(every))
  })
})

describe('the field a goal steers', () => {
  it('is the minutes under a goal of minutes, the target under retention, the day under a date', () => {
    expect(getGoalField('minutes')).toBe('minutesADay')
    expect(getGoalField('retention')).toBe('retention')
    expect(getGoalField('date')).toBe('byDate')
  })
})
