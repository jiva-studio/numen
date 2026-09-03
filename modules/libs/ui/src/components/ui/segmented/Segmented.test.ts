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
  { id: 'small', text: 'Small' },
  { id: 'medium', text: 'Medium' },
  { id: 'large', text: 'Large' },
]

type SegmentedProps = InstanceType<typeof Segmented>['$props']

const mountSegmented = (props: Partial<SegmentedProps> = {}) =>
  mount(Segmented, { props: { choices: CHOICES, modelValue: 'small', ...props } })

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
      'Small',
      'Medium',
      'Large',
    ])
  })
})

describe('choosing', () => {
  it('hands back the identifier it was given', async () => {
    const control = mountSegmented()
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(handed(control)).toEqual(['medium'])
  })

  it('leaves the segment in force in force', async () => {
    const control = mountSegmented()
    await control.findAll('[role="radio"]')[0]?.trigger('click')

    expect(handed(control).at(-1) ?? 'small').toBe('small')
    expect(control.findAll('[role="radio"]')[0]?.attributes('aria-checked')).toBe('true')
  })

  it('goes to the ends on Home and End', async () => {
    const control = mountSegmented({ modelValue: 'medium' })
    const segments = control.findAll('[role="radio"]')

    await segments[1]?.trigger('keydown', { key: 'End' })
    expect(handed(control).at(-1)).toBe('large')

    await segments[1]?.trigger('keydown', { key: 'Home' })
    expect(handed(control).at(-1)).toBe('small')
  })

  it('is left where it is on Home and End while nobody may turn it', async () => {
    const control = mountSegmented({ modelValue: 'medium', disabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('keydown', { key: 'End' })
    expect(handed(control)).toEqual([])
  })

  it('hands nothing on while nobody may turn it', async () => {
    const control = mountSegmented({ disabled: true })
    await control.findAll('[role="radio"]')[1]?.trigger('click')
    expect(handed(control)).toEqual([])
  })
})

// One press is one turn. The row already walks to its ends on these keys, and
// a choice made twice is written twice by whoever is listening.
it('hands a choice on once for one press of Home or End', async () => {
  const control = mountSegmented({ modelValue: 'medium' })
  const segments = control.findAll('[role="radio"]')

  await segments[1]?.trigger('keydown', { key: 'End' })
  expect(handed(control)).toEqual(['large'])
})
