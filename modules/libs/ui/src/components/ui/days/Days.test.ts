/**
 * What the chips are announced as, and that each day is turned on its own.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Days from './Days.vue'
import { weekFrom } from './week'

type DaysProps = InstanceType<typeof Days>['$props']

const mountDays = (props: Partial<DaysProps> = {}) =>
  mount(Days, { props: { modelValue: ['sat'], ...props } })

/** Every list of days the row has handed on, in the order it handed them on. */
const handed = (row: ReturnType<typeof mountDays>): readonly unknown[] =>
  (row.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('what is drawn', () => {
  it('draws the seven days, each under its whole name', () => {
    const chips = mountDays().findAll('button')
    expect(chips).toHaveLength(7)
    expect(chips.map((chip) => chip.attributes('aria-label'))).toEqual([
      'Monday',
      'Tuesday',
      'Wednesday',
      'Thursday',
      'Friday',
      'Saturday',
      'Sunday',
    ])
    expect(chips.map((chip) => chip.text())).toEqual(['M', 'T', 'W', 'T', 'F', 'S', 'S'])
  })

  it('draws the week from the day it is given', () => {
    const chips = mountDays({ days: weekFrom('sun') }).findAll('button')
    expect(chips[0]?.attributes('aria-label')).toBe('Sunday')
  })

  it('says which days are on', () => {
    const chips = mountDays().findAll('button')
    expect(chips.map((chip) => chip.attributes('aria-pressed'))).toEqual([
      'false',
      'false',
      'false',
      'false',
      'false',
      'true',
      'false',
    ])
  })
})

describe('turning a day', () => {
  it('turns it on beside the days already on, in the order the week runs', async () => {
    const row = mountDays()
    await row.findAll('button')[1]?.trigger('click')
    expect(handed(row)).toEqual([['tue', 'sat']])
  })

  it('turns one off and leaves the rest where they were', async () => {
    const row = mountDays({ modelValue: ['tue', 'sat'] })
    await row.findAll('button')[5]?.trigger('click')
    expect(handed(row)).toEqual([['tue']])
  })

  it('hands nothing on while nobody may turn them', async () => {
    const row = mountDays({ disabled: true })
    await row.findAll('button')[1]?.trigger('click')
    expect(handed(row)).toEqual([])
  })
})
