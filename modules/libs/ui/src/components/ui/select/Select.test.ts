/**
 * What a select is announced as, and what it hands back.
 *
 * The identifier the caller gave a choice is the identifier it gets back, and a
 * value in force that is none of the choices leaves the menu standing on none.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Select from './Select.vue'

const CHOICES = [
  { id: 'small', text: 'Small' },
  { id: 'medium', text: 'Medium' },
  { id: 'large', text: 'Large' },
]

type SelectProps = InstanceType<typeof Select>['$props']

const mountSelect = (props: Partial<SelectProps> = {}) =>
  mount(Select, { props: { choices: CHOICES, modelValue: 'small', ...props } })

/** Every choice the control has handed on, in the order it handed them on. */
const handed = (control: ReturnType<typeof mountSelect>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('the choices offered', () => {
  it('are drawn in the order they were given', () => {
    const control = mountSelect()
    expect(control.findAll('option').map((one) => one.text())).toStrictEqual([
      'Small',
      'Medium',
      'Large',
    ])
  })

  it('stand on the shelves they name, in the runs they arrive in', () => {
    const control = mountSelect({
      choices: [
        { id: 'numen', text: 'numen', group: 'Ships with numen' },
        { id: 'sea', text: 'sea', group: 'Yours' },
      ],
    })

    expect(control.findAll('optgroup').map((one) => one.attributes('label'))).toStrictEqual([
      'Ships with numen',
      'Yours',
    ])
  })

  it('are drawn on no shelf where they name none', () => {
    expect(mountSelect().findAll('optgroup')).toHaveLength(0)
  })
})

describe('choosing', () => {
  it('opens on the value in force', () => {
    expect((mountSelect({ modelValue: 'medium' }).get('select').element as HTMLSelectElement).value)
      .toBe('medium')
  })

  it('hands back the identifier it was given', async () => {
    const control = mountSelect()
    await control.get('select').setValue('large')
    expect(handed(control)).toStrictEqual(['large'])
  })

  it('stands on none where the value in force is none of the choices', () => {
    const control = mountSelect({ modelValue: 'enormous' })
    expect((control.get('select').element as HTMLSelectElement).selectedIndex).toBe(-1)
    expect(handed(control)).toStrictEqual([])
  })

  it('is not turned while nobody may turn it', () => {
    expect(mountSelect({ disabled: true }).get('select').attributes('disabled')).toBeDefined()
  })
})
