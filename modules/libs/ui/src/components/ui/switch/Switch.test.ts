/**
 * What the switch is announced as, and what turns it.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Switch from './Switch.vue'

type SwitchProps = InstanceType<typeof Switch>['$props']

const mountSwitch = (props: Partial<SwitchProps> = {}) =>
  mount(Switch, { props: { modelValue: false, ...props } })

/** Every answer the switch has handed on, in the order it handed them on. */
const handed = (control: ReturnType<typeof mountSwitch>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

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
    expect(handed(control)).toEqual([true])
  })

  it('hands nothing on while nobody may turn it', async () => {
    const control = mountSwitch({ disabled: true })
    await control.get('button').trigger('click')
    expect(handed(control)).toEqual([])
  })

  it('is one stop on the way round the screen', () => {
    expect(mountSwitch().get('button').attributes('tabindex')).not.toBe('-1')
  })
})
