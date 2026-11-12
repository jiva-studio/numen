/**
 * What the segments are announced as, and that one of them stays chosen.
 *
 * Choosing the segment already in force leaves it in force: this is a choice
 * between the choices offered, and there is no way to choose none of them.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SegmentedControl from './SegmentedControl.vue'

const CHOICES = [
  { id: 'small', text: 'Small' },
  { id: 'medium', text: 'Medium' },
  { id: 'large', text: 'Large' },
]

type SegmentedControlProps = InstanceType<typeof SegmentedControl>['$props']

const mountSegmentedControl = (props: Partial<SegmentedControlProps> = {}) =>
  mount(SegmentedControl, { props: { choices: CHOICES, modelValue: 'small', ...props } })

const getEmittedEvents = (control: ReturnType<typeof mountSegmentedControl>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((response) => (response as unknown[])[0])

describe('what a screen reader receives', () => {
  it('is a set of choices, one of them in force', () => {
    const control = mountSegmentedControl()
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
    const control = mountSegmentedControl()
    expect(control.findAll('[role="radio"]').map((one) => one.text())).toEqual([
      'Small',
      'Medium',
      'Large',
    ])
  })
})

describe('choosing', () => {
  it('returns the identifier it was given', async () => {
    const control = mountSegmentedControl()
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(getEmittedEvents(control)).toEqual(['medium'])
  })

  it('leaves the segment in force in force', async () => {
    const control = mountSegmentedControl()
    await control.findAll('[role="radio"]')[0]?.trigger('click')

    expect(getEmittedEvents(control).at(-1) ?? 'small').toBe('small')
    expect(control.findAll('[role="radio"]')[0]?.attributes('aria-checked')).toBe('true')
  })

  it('goes to the ends on Home and End', async () => {
    const control = mountSegmentedControl({ modelValue: 'medium' })
    const segments = control.findAll('[role="radio"]')

    await segments[1]?.trigger('keydown', { key: 'End' })
    expect(getEmittedEvents(control).at(-1)).toBe('large')

    await segments[1]?.trigger('keydown', { key: 'Home' })
    expect(getEmittedEvents(control).at(-1)).toBe('small')
  })

  it('is left where it is on Home and End while it cannot be turned', async () => {
    const control = mountSegmentedControl({ modelValue: 'medium', isDisabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('keydown', { key: 'End' })
    expect(getEmittedEvents(control)).toEqual([])
  })

  it('passes nothing on while it cannot be turned', async () => {
    const control = mountSegmentedControl({ isDisabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(getEmittedEvents(control)).toEqual([])
  })
})

// One press is one change. The row already moves to its ends on these keys,
// and a choice made twice would be emitted twice to the listener.
it('passes a choice on once for one press of Home or End', async () => {
  const control = mountSegmentedControl({ modelValue: 'medium' })
  const segments = control.findAll('[role="radio"]')

  await segments[1]?.trigger('keydown', { key: 'End' })
  expect(getEmittedEvents(control)).toEqual(['large'])
})

describe('the other keys', () => {
  it('leave the choice where it is for a key that is not an end', async () => {
    const control = mountSegmentedControl({ modelValue: 'medium' })
    await control.findAll('[role="radio"]')[1]?.trigger('keydown', { key: 'a' })
    expect(getEmittedEvents(control)).toEqual([])
  })
})
