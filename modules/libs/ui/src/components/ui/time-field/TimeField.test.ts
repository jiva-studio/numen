/**
 * What the field hands on.
 *
 * An hour of the day is handed on once; anything that is not one is left where
 * it stands and nobody is told.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TimeField from './TimeField.vue'

type TimeFieldProps = InstanceType<typeof TimeField>['$props']

const mountField = (props: Partial<TimeFieldProps> = {}) =>
  mount(TimeField, { props: { modelValue: '04:00', ...props } })

/** Every hour the field has handed on, in the order it handed them on. */
const handed = (field: ReturnType<typeof mountField>): readonly unknown[] =>
  (field.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('the hour in force', () => {
  it('stands in the field as it was given', () => {
    expect((mountField().get('input').element as HTMLInputElement).value).toBe('04:00')
  })

  it('stands at nothing where what was given is no hour of the day', () => {
    expect((mountField({ modelValue: 'noon' }).get('input').element as HTMLInputElement).value)
      .toBe('')
  })
})

describe('an hour typed', () => {
  it('is handed on, and said to have settled', async () => {
    const field = mountField()
    await field.get('input').setValue('06:30')
    expect(handed(field)).toStrictEqual(['06:30'])
    expect(field.emitted('settles')).toStrictEqual([['06:30']])
  })

  it('is handed on once where it is the hour already in force', async () => {
    const field = mountField()
    await field.get('input').setValue('04:00')
    expect(handed(field)).toStrictEqual([])
  })

  it('is left where it stands where it is no hour of the day', async () => {
    const field = mountField()
    await field.get('input').setValue('')
    expect(handed(field)).toStrictEqual([])
    expect(field.emitted('settles')).toBeUndefined()
  })
})

describe('the ends of the day', () => {
  it('are as early and as late as the caller says', () => {
    const field = mountField({ min: '00:00', max: '12:00' })
    expect(field.get('input').attributes('min')).toBe('00:00')
    expect(field.get('input').attributes('max')).toBe('12:00')
  })

  it('are the whole day where the caller says nothing', () => {
    const field = mountField()
    expect(field.get('input').attributes('min')).toBeUndefined()
    expect(field.get('input').attributes('max')).toBeUndefined()
  })
})

it('is not typed into while nobody may turn it', () => {
  expect(mountField({ disabled: true }).get('input').attributes('disabled')).toBeDefined()
})
