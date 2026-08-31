/**
 * What the segments are announced as, and that one of them stays chosen.
 *
 * Choosing the segment already in force leaves it in force: this is a choice
 * between the choices offered, and there is no way to choose none of them.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Segmented from './Segmented.vue'

const CHOICES = [
  { id: 'minutes', text: 'Minutes a day' },
  { id: 'retention', text: 'Retention' },
  { id: 'date', text: 'By a date' },
]

type SegmentedProps = InstanceType<typeof Segmented>['$props']

const mountSegmented = (props: Partial<SegmentedProps> = {}) =>
  mount(Segmented, { props: { choices: CHOICES, modelValue: 'minutes', ...props } })

/** Every choice the control has handed on, in the order it handed them on. */
const handed = (control: ReturnType<typeof mountSegmented>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('what a screen reader is told', () => {
  it('is a set of choices, one of them in force', () => {
    const control = mountSegmented()
    expect(control.get('[role="radiogroup"]')).toBeTruthy()

    const segments = control.findAll('[role="radio"]')
    expect(segments).toHaveLength(3)
    expect(segments.map((one) => one.attributes('aria-checked'))).toEqual([
      'true',
      'false',
      'false',
    ])
  })

  it('draws what each choice says, in the order they were offered', () => {
    const control = mountSegmented()
    expect(control.findAll('[role="radio"]').map((one) => one.text())).toEqual([
      'Minutes a day',
      'Retention',
      'By a date',
    ])
  })
})

describe('choosing', () => {
  it('hands back the identifier it was given', async () => {
    const control = mountSegmented()
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(handed(control)).toEqual(['retention'])
  })

  it('leaves the segment in force in force', async () => {
    const control = mountSegmented()
    await control.findAll('[role="radio"]')[0]?.trigger('click')

    expect(handed(control).at(-1) ?? 'minutes').toBe('minutes')
    expect(control.findAll('[role="radio"]')[0]?.attributes('aria-checked')).toBe('true')
  })

  it('goes to the ends on Home and End', async () => {
    const control = mountSegmented({ modelValue: 'retention' })
    const segments = control.findAll('[role="radio"]')

    await segments[1]?.trigger('keydown', { key: 'End' })
    expect(handed(control).at(-1)).toBe('date')

    await segments[1]?.trigger('keydown', { key: 'Home' })
    expect(handed(control).at(-1)).toBe('minutes')
  })

  it('is left where it is on Home and End while nobody may turn it', async () => {
    const control = mountSegmented({ modelValue: 'retention', disabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('keydown', { key: 'End' })
    expect(handed(control)).toEqual([])
  })

  it('hands nothing on while nobody may turn it', async () => {
    const control = mountSegmented({ disabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(handed(control)).toEqual([])
  })
})
