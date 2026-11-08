/**
 * What the checkbox emits, when it does not toggle, and how it gets its name.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Checkbox from './Checkbox.vue'

const getEmitted = (box: ReturnType<typeof mount>): readonly unknown[] =>
  (box.emitted('update:modelValue') ?? []).map((args) => (args as unknown[])[0])

describe('Checkbox', () => {
  it('is a checkbox input that shows the value it is given', () => {
    const box = mount(Checkbox, { props: { modelValue: true }, attrs: { 'aria-label': 'Read' } })
    const input = box.get('input')
    expect(input.attributes('type')).toBe('checkbox')
    expect((input.element as HTMLInputElement).checked).toBe(true)
  })

  it('emits the new value when it is pressed', async () => {
    const box = mount(Checkbox, { props: { modelValue: false } })
    await box.get('input').setValue(true)
    expect(getEmitted(box)).toEqual([true])
  })

  it('emits false when a ticked box is pressed', async () => {
    const box = mount(Checkbox, { props: { modelValue: true } })
    await box.get('input').setValue(false)
    expect(getEmitted(box)).toEqual([false])
  })

  it('does not toggle when disabled', async () => {
    const box = mount(Checkbox, { props: { modelValue: false, isDisabled: true } })
    const input = box.get('input')
    expect(input.attributes('disabled')).toBeDefined()
    await input.trigger('click')
    expect(getEmitted(box)).toEqual([])
  })

  it('passes aria-label to the input', () => {
    const box = mount(Checkbox, { attrs: { 'aria-label': 'Allow' } })
    expect(box.get('input').attributes('aria-label')).toBe('Allow')
  })

  it('is named by the slot through a wrapping label', () => {
    const box = mount(Checkbox, { slots: { default: 'Allow reads' } })
    const label = box.get('label')
    expect(label.text()).toBe('Allow reads')
    expect(label.find('input').exists()).toBe(true)
  })

  it('toggles when the label text is clicked', async () => {
    const box = mount(Checkbox, {
      props: { modelValue: false },
      slots: { default: 'Allow reads' },
      attachTo: document.body,
    })
    await box.get('label').trigger('click')
    expect(getEmitted(box)).toEqual([true])
    box.unmount()
  })
})
