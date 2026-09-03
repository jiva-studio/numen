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

// A caller narrowing what a field allows narrows it under a number already
// standing there. Nobody typed that number here, so nothing is marked.
describe('the bounds moving under the number', () => {
  it('brings the number in, writes it out, and marks nothing', async () => {
    const field = mountField({ modelValue: 200 })
    await field.setProps({ max: 10 })

    expect(handed(field)).toEqual([10])
    expect(field.get('input').element.value).toBe('10')
    expect(field.get('input').attributes('aria-valuenow')).toBe('10')
    expect(field.get('input').attributes('aria-invalid')).toBeUndefined()
  })

  it('brings a number under a floor that rose up to it', async () => {
    const field = mountField({ modelValue: 20 })
    await field.setProps({ min: 100 })

    expect(handed(field)).toEqual([100])
    expect(field.get('input').element.value).toBe('100')
  })

  it('brings in a number the bounds it opened on never held', () => {
    const field = mountField({ modelValue: 900 })
    expect(field.get('input').element.value).toBe('240')
    expect(handed(field)).toEqual([240])
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

// A spin button answers six keys, and a field answering two of them is a
// control the keyboard cannot reach the ends of.
describe('the other keys a spin button answers', () => {
  it('takes the number to the ends under home and end', async () => {
    const field = mountField()
    await field.get('input').trigger('keydown', { key: 'End' })
    expect(field.get('input').element.value).toBe('240')

    await field.get('input').trigger('keydown', { key: 'Home' })
    expect(field.get('input').element.value).toBe('0')
    expect(handed(field)).toEqual([240, 0])
  })

  it('moves ten steps under the page keys, and stops at the bounds', async () => {
    const field = mountField({ modelValue: 100 })
    await field.get('input').trigger('keydown', { key: 'PageDown' })
    expect(field.get('input').element.value).toBe('50')

    await field.get('input').trigger('keydown', { key: 'PageUp' })
    expect(field.get('input').element.value).toBe('100')

    await field.get('input').trigger('keydown', { key: 'PageDown' })
    await field.get('input').trigger('keydown', { key: 'PageDown' })
    await field.get('input').trigger('keydown', { key: 'PageDown' })
    expect(field.get('input').element.value).toBe('0')
  })

  it('leaves a field nobody may type into where it stands', async () => {
    const field = mountField({ disabled: true })
    await field.get('input').trigger('keydown', { key: 'End' })
    await field.get('input').trigger('keydown', { key: 'PageUp' })
    expect(handed(field)).toEqual([])
  })

  it('leaves a key it does not answer to the field it is typed into', async () => {
    const field = mountField()
    await field.get('input').trigger('keydown', { key: 'a' })
    expect(handed(field)).toEqual([])
    expect(field.get('input').element.value).toBe('20')
  })
})

// A caller writes what it holds when the field settles, so a field left alone
// is a field that has settled nowhere.
describe('leaving the field', () => {
  /** Every number the field has come to rest at, in the order it rested at them. */
  const rested = (field: ReturnType<typeof mountField>): readonly unknown[] =>
    (field.emitted('settles') ?? []).map((said) => (said as unknown[])[0])

  it('says nothing where nothing was typed into it', async () => {
    const field = mountField()
    await field.get('input').trigger('blur')
    await field.get('input').trigger('focus')
    await field.get('input').trigger('blur')

    expect(rested(field)).toEqual([])
  })

  it('says nothing where what was typed came to the number already standing', async () => {
    const field = mountField()
    await field.get('input').setValue('20')
    await field.get('input').trigger('blur')

    expect(rested(field)).toEqual([])
  })

  it('says it where the number the field comes to rest at is another one', async () => {
    const field = mountField()
    await field.get('input').setValue('45')
    await field.get('input').trigger('blur')
    await field.get('input').trigger('blur')

    expect(rested(field)).toEqual([45])
  })

  it('says nothing where a number set from outside is the one it is left at', async () => {
    const field = mountField()
    await field.setProps({ modelValue: 60 })
    await field.get('input').trigger('blur')

    expect(rested(field)).toEqual([])
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

  it('says nothing of its own where the number in force is what stands there', () => {
    expect(mountField().get('input').attributes('aria-valuetext')).toBeUndefined()
  })

  // A line refused with no number to announce is refused in silence, and the
  // eye can see what the ear is not told.
  it('reads out a line that is no number, where no number stands with it', async () => {
    const field = mountField({ modelValue: null })
    await field.get('input').setValue('twenty')
    const input = field.get('input')

    expect(input.attributes('aria-invalid')).toBe('true')
    expect(input.attributes('aria-valuenow')).toBeUndefined()
    expect(input.attributes('aria-valuetext')).toBe('twenty')
  })

  it('reads out a number past the bounds, which are said beside it', async () => {
    const field = mountField()
    await field.get('input').setValue('900')
    const input = field.get('input')

    expect(input.attributes('aria-valuetext')).toBe('900')
    expect(input.attributes('aria-valuemax')).toBe('240')
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

  // A line refused stands for no number, so a field emptied from outside is
  // emptied whatever is standing in it.
  it('empties the field where the number is taken away under refused text', async () => {
    const field = mountField()
    await field.get('input').setValue('twenty')
    await field.setProps({ modelValue: null })

    expect(field.get('input').element.value).toBe('')
    expect(field.get('input').attributes('aria-invalid')).toBeUndefined()
    expect(field.get('input').attributes('aria-valuetext')).toBeUndefined()
  })
})

/**
 * A field whose caller holds the value and may refuse one.
 *
 * Passing the value and the listener both leaves the value the caller's: what
 * the field hands on is a request, and what stands in the field is whatever the
 * caller holds after it.
 */
describe('a field the caller holds the value of', () => {
  const held = (refuses: (said: number | null) => boolean) => {
    let standing: number | null = 20
    const field = mount(NumberField, {
      props: {
        modelValue: standing,
        min: 0,
        max: 240,
        step: 5,
        'onUpdate:modelValue': async (said: number | null) => {
          if (refuses(said)) return
          standing = said
          await field.setProps({ modelValue: standing })
        },
      },
    })
    return field
  }

  it('stands on the number the caller holds where the caller refuses one', async () => {
    const field = held((said) => said === null)
    await field.get('input').setValue('')
    await field.get('input').trigger('blur')
    await field.vm.$nextTick()

    expect(field.get('input').element.value).toBe('20')
  })

  it('stands on the number the caller took where the caller took one', async () => {
    const field = held((said) => said === null)
    await field.get('input').setValue('45')
    await field.get('input').trigger('blur')
    await field.vm.$nextTick()

    expect(field.get('input').element.value).toBe('45')
  })
})
