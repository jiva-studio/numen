import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ReasoningSelector from './ReasoningSelector.vue'
import type { ReasoningLevelOption } from './ReasoningSelector.vue'

const LEVELS: ReasoningLevelOption[] = [
  { id: 'low', title: 'Low' },
  { id: 'medium', title: 'Medium' },
  { id: 'high', title: 'High' },
]

describe('ReasoningSelector', () => {
  it('shows the title of the chosen level', () => {
    const wrapper = mount(ReasoningSelector, { props: { levels: LEVELS, selectedId: 'high' } })
    expect(wrapper.text()).toContain('High')
  })

  it('shows its name while no level matches the choice', () => {
    const wrapper = mount(ReasoningSelector, {
      props: { levels: LEVELS, selectedId: '', name: 'Thinking depth' },
    })
    expect(wrapper.text()).toContain('Thinking depth')
    expect(wrapper.get('button').attributes('aria-label')).toBe('Thinking depth')
  })

  it('is disabled with no levels, and when told so', () => {
    const none = mount(ReasoningSelector, { props: { levels: [] } })
    expect(none.get('button').attributes('disabled')).toBeDefined()
    const off = mount(ReasoningSelector, { props: { levels: LEVELS, disabled: true } })
    expect(off.get('button').attributes('disabled')).toBeDefined()
  })

  it('opens its menu with the levels and says the one chosen', async () => {
    const wrapper = mount(ReasoningSelector, {
      props: { levels: LEVELS, selectedId: 'medium' },
      attachTo: document.body,
    })
    await wrapper.get('button').trigger('click')

    const menu = wrapper.findComponent({ name: 'Menu' })
    expect(menu.props('open')).toBe(true)
    expect(menu.props('current')).toBe('medium')
    expect((menu.props('items') as { id: string }[]).map((one) => one.id)).toEqual([
      'low',
      'medium',
      'high',
    ])

    await menu.vm.$emit('choose', 'low')
    expect(wrapper.emitted('select')).toEqual([['low']])
    expect(menu.props('open')).toBe(false)
    wrapper.unmount()
  })

  it('ignores a choice that is no level', async () => {
    const wrapper = mount(ReasoningSelector, {
      props: { levels: LEVELS },
      attachTo: document.body,
    })
    await wrapper.get('button').trigger('click')
    await wrapper.findComponent({ name: 'Menu' }).vm.$emit('choose', 'bogus')
    expect(wrapper.emitted('select')).toBeUndefined()
    wrapper.unmount()
  })

  it('closes the menu when it is dismissed', async () => {
    const wrapper = mount(ReasoningSelector, {
      props: { levels: LEVELS },
      attachTo: document.body,
    })
    await wrapper.get('button').trigger('click')
    const menu = wrapper.findComponent({ name: 'Menu' })
    await menu.vm.$emit('dismiss')
    expect(menu.props('open')).toBe(false)
    wrapper.unmount()
  })
})
