/**
 * What a rule draws around what stands in its middle.
 *
 * The negatives are here: it is announced as nothing, it draws no line of its
 * own as an element, and it takes no name away from what it holds.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it } from 'vitest'
import Rule from './Rule.vue'

const held = (inside: string) =>
  mount(Rule, {
    slots: { default: h('button', { type: 'button' }, inside) },
  })

describe('Rule', () => {
  it('stands what it holds in its middle, and nothing else', () => {
    const rule = held('Add a field')
    expect(rule.element.children).toHaveLength(1)
    expect(rule.get('button').text()).toBe('Add a field')
  })

  it('is announced as nothing, so nothing is said around what it holds', () => {
    const rule = held('Add a field')
    expect(rule.attributes('role')).toBe('presentation')
    expect(rule.attributes('aria-orientation')).toBeUndefined()
  })

  it('draws its line as decoration, and never as an element', () => {
    const rule = held('Add a field')
    expect(rule.element.tagName).toBe('DIV')
    expect(rule.findAll('hr')).toHaveLength(0)
  })

  it('stands what it holds in its middle where it is asked for nothing else', () => {
    expect(held('Add a field').attributes('data-at')).toBe('middle')
  })

  it('leads with what it holds where it is asked to', () => {
    const rule = mount(Rule, {
      props: { at: 'start' },
      slots: { default: h('button', { type: 'button' }, 'Turn') },
    })
    expect(rule.attributes('data-at')).toBe('start')
    expect(rule.get('button').text()).toBe('Turn')
  })

  it('leaves what it holds a button of its own, under its own name', () => {
    const rule = held('Add a face')
    const button = rule.get('button')
    expect(button.attributes('role')).toBeUndefined()
    expect(button.attributes('aria-hidden')).toBeUndefined()
    expect(button.element.tagName).toBe('BUTTON')
  })

  it('holds whatever it is handed', () => {
    const other = defineComponent({ setup: () => () => h('span', 'said') })
    const rule = mount(Rule, { slots: { default: h(other) } })
    expect(rule.get('span').text()).toBe('said')
  })
})
