/**
 * What the switch is announced as, and what turns it.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Switch from './Switch.vue'

type SwitchProps = InstanceType<typeof Switch>['$props']

const mountSwitch = (props: Partial<SwitchProps> = {}) =>
  mount(Switch, { props: { modelValue: false, ...props } })

const getEmittedEvents = (control: ReturnType<typeof mountSwitch>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((response) => (response as unknown[])[0])

describe('Switch', () => {
  it('is announced as a switch, and says which way it is', () => {
    const control = mountSwitch({ modelValue: true })
    expect(control.get('button').attributes('role')).toBe('switch')
    expect(control.get('button').attributes('aria-checked')).toBe('true')
  })

  it('says it is off where it is off', () => {
    expect(mountSwitch().get('button').attributes('aria-checked')).toBe('false')
  })

  it('turns the other way when it is pressed', async () => {
    const control = mountSwitch()
    await control.get('button').trigger('click')
    expect(getEmittedEvents(control)).toEqual([true])
  })

  it('passes nothing on while it cannot be turned', async () => {
    const control = mountSwitch({ isDisabled: true })
    await control.get('button').trigger('click')
    expect(getEmittedEvents(control)).toEqual([])
  })

  it('keeps its look when pressed, until the parent passes the new value', async () => {
    const control = mountSwitch({ modelValue: false })
    await control.get('button').trigger('click')
    expect(getEmittedEvents(control)).toEqual([true])
    expect(control.get('button').attributes('aria-checked')).toBe('false')
    expect(control.get('button').attributes('data-state')).toBe('unchecked')
    await control.setProps({ modelValue: true })
    expect(control.get('button').attributes('aria-checked')).toBe('true')
  })

  it('ignores presses and is not drawn disabled while a change is being written', async () => {
    const control = mountSwitch({ modelValue: true, isBusy: true })
    await control.get('button').trigger('click')
    await control.get('button').trigger('keydown', { key: 'Enter' })
    expect(getEmittedEvents(control)).toEqual([])
    const button = control.get('button')
    expect(button.attributes('disabled')).toBeUndefined()
    expect(button.attributes('data-disabled')).toBeUndefined()
    expect(button.attributes('aria-busy')).toBe('true')
    expect(button.attributes('aria-checked')).toBe('true')
  })

  it('is one stop on the way round the screen', () => {
    expect(mountSwitch().get('button').attributes('tabindex')).not.toBe('-1')
  })
})
