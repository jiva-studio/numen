/**
 * What the field hands on.
 *
 * An hour of the day is handed on once; anything that is not one is left where
 * it stands and nobody is told.
 */
import { mount } from '@vue/test-utils'
import { h } from 'vue'
import { describe, expect, it } from 'vitest'
import TimeField from './TimeField.vue'

type TimeFieldProps = InstanceType<typeof TimeField>['$props']

const mountField = (props: Partial<TimeFieldProps> = {}) =>
  mount(TimeField, { props: { modelValue: '04:00', ...props } })

/** Every hour the field has handed on, in the order it handed them on. */
const getEmitted = (field: ReturnType<typeof mountField>): readonly unknown[] =>
  (field.emitted('update:modelValue') ?? []).map((response) => (response as unknown[])[0])

describe('the hour in force', () => {
  it('remains in the field as it was given', () => {
    expect((mountField().get('input').element as HTMLInputElement).value).toBe('04:00')
  })

  it('remains at nothing where what was given is no hour of the day', () => {
    expect(
      (mountField({ modelValue: 'noon' }).get('input').element as HTMLInputElement).value,
    ).toBe('')
  })
})

describe('an hour typed', () => {
  it('is handed on, and returned to have committed', async () => {
    const field = mountField()
    await field.get('input').setValue('06:30')
    expect(getEmitted(field)).toStrictEqual(['06:30'])
    expect(field.emitted('commit')).toStrictEqual([['06:30']])
  })

  it('is handed on once where it is the hour already in force', async () => {
    const field = mountField()
    await field.get('input').setValue('04:00')
    expect(getEmitted(field)).toStrictEqual([])
  })

  it('is left at its current position where it is no hour of the day', async () => {
    const field = mountField()
    await field.get('input').setValue('')
    expect(getEmitted(field)).toStrictEqual([])
    expect(field.emitted('commit')).toBeUndefined()
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

describe('what the field is called', () => {
  it('is named by a label pointing at the id it is given', () => {
    const named = mount(
      () => [
        h('label', { for: 'day-begins' }, 'A day begins at'),
        h(TimeField, { id: 'day-begins' }),
      ],
      { attachTo: document.body },
    )
    const field = named.get<HTMLInputElement>('input').element

    expect([...(field.labels ?? [])].map((label) => label.textContent)).toEqual(['A day begins at'])
    named.unmount()
  })

  it('makes up an id of its own where it is given none, and no two alike', () => {
    const pair = mount(() => h('div', [h(TimeField, {}), h(TimeField, {})]))
    const ids = pair.findAll('input').map((field) => field.attributes('id'))

    expect(ids.every(Boolean)).toBe(true)
    expect(new Set(ids).size).toBe(2)
  })
})
