/**
 * What the field does with what is typed into it.
 *
 * The negatives matter most: a line that is not a number is not swallowed and
 * is not handed on, and a number past the bounds is neither.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NumberField from './NumberField.vue'

type FieldProps = InstanceType<typeof NumberField>['$props']

const mountField = (props: Partial<FieldProps> = {}) =>
  mount(NumberField, {
    props: { modelValue: 20, min: 0, max: 240, step: 5, ...props },
  })

/** Every number the field has handed on, in the order it handed them on. */
const handed = (field: ReturnType<typeof mountField>): readonly unknown[] =>
  (field.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('what is typed', () => {
  it('stands as it was typed while it is being typed', async () => {
    const field = mountField()
    await field.get('input').setValue('12x')

    expect(field.get('input').element.value).toBe('12x')
    expect(handed(field)).toEqual([])
  })

  it('is marked where nothing further would make a number of it', async () => {
    const field = mountField()
    await field.get('input').setValue('twenty')
    expect(field.get('input').attributes('aria-invalid')).toBe('true')
  })

  it('is left unmarked while a number could still be typed out of it', async () => {
    const field = mountField()
    await field.get('input').setValue('1.')
    expect(field.get('input').attributes('aria-invalid')).toBeUndefined()
  })

  it('is handed on as soon as it is a number the bounds allow', async () => {
    const field = mountField()
    await field.get('input').setValue('45')
    expect(handed(field)).toEqual([45])
  })

  it('holds no number at all once the field is emptied', async () => {
    const field = mountField()
    await field.get('input').setValue('')
    expect(handed(field)).toEqual([null])
  })

  it('marks a thousands group and hands nothing on', async () => {
    const field = mountField()
    await field.get('input').setValue('1,200')

    expect(field.get('input').attributes('aria-invalid')).toBe('true')
    expect(handed(field)).toEqual([])
  })
})

describe('a number between two places the step lays', () => {
  it('hands nothing on while it stands there', async () => {
    const field = mountField()
    await field.get('input').setValue('13')

    expect(field.get('input').element.value).toBe('13')
    expect(handed(field)).toEqual([])
  })

  it('is brought onto a place of the step once the field is left', async () => {
    const field = mountField()
    await field.get('input').setValue('13')
    await field.get('input').trigger('blur')

    expect(handed(field)).toEqual([15])
    expect(field.get('input').element.value).toBe('15')
  })

  it('leaves an arrow key standing on a place of the step', async () => {
    const field = mountField({ modelValue: 10 })
    await field.get('input').setValue('13')
    await field.get('input').trigger('keydown', { key: 'ArrowUp' })

    expect(handed(field)).toEqual([20])
    expect(field.get('input').element.value).toBe('20')
  })
})

describe('the bounds', () => {
  it('marks a number past them and hands nothing on', async () => {
    const field = mountField()
    await field.get('input').setValue('900')

    expect(field.get('input').element.value).toBe('900')
    expect(field.get('input').attributes('aria-invalid')).toBe('true')
    expect(handed(field)).toEqual([])
  })

  it('brings what stands there inside them once the field is left', async () => {
    const field = mountField()
    await field.get('input').setValue('900')
    await field.get('input').trigger('blur')

    expect(handed(field)).toEqual([240])
    expect(field.get('input').element.value).toBe('240')
    expect(field.get('input').attributes('aria-invalid')).toBeUndefined()
  })

  it('leaves no line that is not a number standing once the field is left', async () => {
    const field = mountField()
    await field.get('input').setValue('twenty')
    await field.get('input').trigger('blur')

    expect(field.get('input').element.value).toBe('')
    expect(handed(field)).toEqual([null])
  })
})

describe('the arrow keys', () => {
  it('move the number one step', async () => {
    const field = mountField()
    await field.get('input').trigger('keydown', { key: 'ArrowUp' })
    expect(handed(field)).toEqual([25])
    expect(field.get('input').element.value).toBe('25')
  })

  it('stop at the bounds', async () => {
    const field = mountField({ modelValue: 2 })
    await field.get('input').trigger('keydown', { key: 'ArrowDown' })
    await field.get('input').trigger('keydown', { key: 'ArrowDown' })

    expect(handed(field)).toEqual([0])
    expect(field.get('input').element.value).toBe('0')
  })

  it('leave a number written to the places its step is written to', async () => {
    const field = mountField({ modelValue: 0.81, min: 0.7, max: 0.99, step: 0.01 })
    await field.get('input').trigger('keydown', { key: 'ArrowUp' })

    expect(handed(field)).toEqual([0.82])
    expect(field.get('input').element.value).toBe('0.82')
  })

  it('leave a field nobody may type into where it stands', async () => {
    const field = mountField({ disabled: true })
    await field.get('input').trigger('keydown', { key: 'ArrowUp' })
    expect(handed(field)).toEqual([])
  })
})

describe('what a screen reader is told', () => {
  it('is a number with bounds, and where in them it stands', () => {
    const input = mountField().get('input')
    expect(input.attributes('role')).toBe('spinbutton')
    expect(input.attributes('aria-valuemin')).toBe('0')
    expect(input.attributes('aria-valuemax')).toBe('240')
    expect(input.attributes('aria-valuenow')).toBe('20')
  })

  it('says no number stands there where none does', () => {
    const input = mountField({ modelValue: null }).get('input')
    expect(input.attributes('aria-valuenow')).toBeUndefined()
    expect(input.element.value).toBe('')
  })
})

describe('a number set from outside', () => {
  it('is written into the field', async () => {
    const field = mountField()
    await field.setProps({ modelValue: 60 })
    expect(field.get('input').element.value).toBe('60')
  })

  it('leaves what is being typed alone where it means that number already', async () => {
    const field = mountField()
    await field.get('input').setValue('60.0')
    await field.setProps({ modelValue: 60 })
    expect(field.get('input').element.value).toBe('60.0')
  })
})
