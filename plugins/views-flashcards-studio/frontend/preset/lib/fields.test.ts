/**
 * Which settings a goal schedules by, and the one word that names their shape.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, type Settings } from '../types'
import {
  FIELDS,
  getFieldCount,
  getFieldLimits,
  getFieldValue,
  getGoalFields,
  getShapeKey,
  getGoalField,
} from './fields'

const settings = (overrides: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...overrides })

describe('the settings a goal schedules by', () => {
  // A goal names one budget, and the budgets of the other two take no part in
  // it: they are not drawn, so nothing on the screen offers to cut the day
  // short by a measure the user did not choose.
  it('is the minutes alone under a goal of minutes', () => {
    const goalFields = getGoalFields('minutes', DEFAULTS.learned)
    expect(goalFields).toContain('minutesADay')
    expect(goalFields).not.toContain('newADay')
    expect(goalFields).not.toContain('reviewsADay')
    expect(goalFields).not.toContain('retention')
    expect(goalFields).not.toContain('byDate')
  })

  it('is the target and the counts that close a day under a goal of retention', () => {
    const goalFields = getGoalFields('retention', DEFAULTS.learned)
    expect(goalFields).toContain('retention')
    expect(goalFields).toContain('newADay')
    expect(goalFields).toContain('reviewsADay')
    expect(goalFields).not.toContain('minutesADay')
    expect(goalFields).not.toContain('byDate')
  })

  it('is the day alone under a goal of a date, which nothing else may cut short', () => {
    const goalFields = getGoalFields('date', DEFAULTS.learned)
    expect(goalFields).toContain('byDate')
    expect(goalFields).not.toContain('minutesADay')
    expect(goalFields).not.toContain('newADay')
    expect(goalFields).not.toContain('reviewsADay')
  })

  it('draws what is placed under no goal in particular under all three', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(getGoalFields(goal, DEFAULTS.learned)).toContain('load')
      expect(getGoalFields(goal, DEFAULTS.learned)).toContain('hasEvenLoad')
    }
  })

  // What a day's budget is spent on is a measure in cards, so it is placed under
  // the goal whose budget is cards and takes no part in the other two.
  it('draws what a budget is spent on under the goal whose budget is cards', () => {
    expect(getGoalFields('retention', DEFAULTS.learned)).toContain('counts')
    expect(getGoalFields('minutes', DEFAULTS.learned)).not.toContain('counts')
    expect(getGoalFields('date', DEFAULTS.learned)).not.toContain('counts')
  })

  // The share of a day that goes to the debt decides what a day is spent on and
  // closes nothing. A goal of a date covers the whole material by its own
  // computation and ignores it.
  it('draws the backlog share under the two goals that keep a budget of a day', () => {
    expect(getGoalFields('minutes', DEFAULTS.learned)).toContain('backlog')
    expect(getGoalFields('retention', DEFAULTS.learned)).toContain('backlog')
    expect(getGoalFields('date', DEFAULTS.learned)).not.toContain('backlog')
  })

  it('draws them in the order the receipt keeps them in', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const goalFields = getGoalFields(goal, DEFAULTS.learned)
      const places = goalFields.map((field) => FIELDS.indexOf(field))
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

  it('remains the rule over the value it reads', () => {
    for (const learned of ['interval', 'retention'] as const) {
      const goalFields = getGoalFields('minutes', learned)
      const value = learned === 'interval' ? 'interval' : 'retention'
      expect(goalFields.indexOf(value)).toBe(goalFields.indexOf('learned') + 1)
    }
  })

  // A curve is recomputed when a setting it is drawn from changes, and both of
  // these change it: what counts as learned is computed from the run.
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
        'minutesADay=20 byDate=2026-09-30 counts=cards backlog=100 load=mon=25,tue=50 hasEvenLoad=true',
    )
  })

  it('leaves the day out under a goal that does not getGoalField by it', () => {
    expect(getShapeKey({ ...every, goal: 'minutes' })).toBe(
      'minutes newADay=10 reviewsADay=200 learned=interval interval=21 retention=0.9 ' +
        'minutesADay=20 counts=cards backlog=100 load=mon=25,tue=50 hasEvenLoad=true',
    )
  })

  it('differs where the reviews of a day, the backlog or the even load differ', () => {
    const shape = getShapeKey(every)

    expect(getShapeKey({ ...every, reviewsADay: 300 })).not.toBe(shape)
    expect(getShapeKey({ ...every, backlog: 50 })).not.toBe(shape)
    expect(getShapeKey({ ...every, hasEvenLoad: false })).not.toBe(shape)
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

describe('the limits of a control', () => {
  const bounds = {
    newADay: { min: 0, max: 50 },
    retention: { min: 0.7, max: 0.99 },
    backlog: { min: 0, max: 100 },
    reviewsADay: { min: 10, max: 200 },
    minutesADay: { min: 5, max: 120 },
  }

  it('are the bounds of the setting', () => {
    expect(getFieldLimits('newADay', bounds)).toStrictEqual({ min: 0, max: 50 })
    expect(getFieldLimits('backlog', bounds)).toStrictEqual({ min: 0, max: 100 })
    expect(getFieldLimits('reviewsADay', bounds)).toStrictEqual({ min: 10, max: 200 })
    expect(getFieldLimits('minutesADay', bounds)).toStrictEqual({ min: 5, max: 120 })
  })

  it('are in percent for retention, which is retained as a share', () => {
    const { min, max } = getFieldLimits('retention', bounds) as { min: number; max: number }
    expect(min).toBeCloseTo(70)
    expect(max).toBeCloseTo(99)
  })

  it('are none for a field with no bounds given, or with no number to bound', () => {
    expect(getFieldLimits('interval', {})).toStrictEqual({})
    expect(getFieldLimits('byDate', bounds)).toStrictEqual({})
    expect(getFieldLimits('load', bounds)).toStrictEqual({})
  })
})

describe('the number a control shows', () => {
  it('is the setting, and a percentage for retention', () => {
    const target = settings({
      newADay: 7,
      retention: 0.9,
      interval: 30,
      reviewsADay: 50,
      minutesADay: 25,
    })
    expect(getFieldCount('newADay', target)).toBe(7)
    expect(getFieldCount('interval', target)).toBe(30)
    expect(getFieldCount('retention', target)).toBe(90)
    expect(getFieldCount('reviewsADay', target)).toBe(50)
    expect(getFieldCount('minutesADay', target)).toBe(25)
  })

  it('is nothing for a field that is no number', () => {
    expect(getFieldCount('byDate', settings())).toBeNull()
    expect(getFieldCount('hasEvenLoad', settings())).toBeNull()
  })
})

describe('the value a number typed evaluates to', () => {
  it('is the number itself, and a share of one for retention', () => {
    expect(getFieldValue('newADay', 12)).toBe(12)
    expect(getFieldValue('retention', 90)).toBe(0.9)
    expect(getFieldValue('retention', 87)).toBe(0.87)
  })

  it('is read back as the percentage it was typed as', () => {
    for (const typed of [70, 85, 93, 99]) {
      expect(
        getFieldCount('retention', settings({ retention: getFieldValue('retention', typed) })),
      ).toBe(typed)
    }
  })
})
