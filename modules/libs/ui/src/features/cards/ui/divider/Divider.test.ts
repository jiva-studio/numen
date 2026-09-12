/**
 * What a divider draws around what stands in its middle.
 *
 * The negatives are here: it is announced as nothing, it draws no line of its
 * own as an element, and it takes no name away from what it holds.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it } from 'vitest'
import Divider from './Divider.vue'

const mountDivider = (inside: string) =>
  mount(Divider, {
    slots: { default: h('button', { type: 'button' }, inside) },
  })

describe('Divider', () => {
  it('stands what it holds in its middle, and nothing else', () => {
    const divider = mountDivider('Add a field')
    expect(divider.element.children).toHaveLength(1)
    expect(divider.get('button').text()).toBe('Add a field')
  })

  it('is announced as nothing, so nothing is said around what it holds', () => {
    const divider = mountDivider('Add a field')
    expect(divider.attributes('role')).toBe('presentation')
    expect(divider.attributes('aria-orientation')).toBeUndefined()
  })

  it('draws its line as decoration, and never as an element', () => {
    const divider = mountDivider('Add a field')
    expect(divider.element.tagName).toBe('DIV')
    expect(divider.findAll('hr')).toHaveLength(0)
  })

  it('stands what it holds in its middle where it is asked for nothing else', () => {
    expect(mountDivider('Add a field').attributes('data-at')).toBe('middle')
  })

  it('leads with what it holds where it is asked to', () => {
    const divider = mount(Divider, {
      props: { at: 'start' },
      slots: { default: h('button', { type: 'button' }, 'Turn') },
    })
    expect(divider.attributes('data-at')).toBe('start')
    expect(divider.get('button').text()).toBe('Turn')
  })

  it('leaves what it holds a button of its own, under its own name', () => {
    const divider = mountDivider('Add a face')
    const button = divider.get('button')
    expect(button.attributes('role')).toBeUndefined()
    expect(button.attributes('aria-hidden')).toBeUndefined()
    expect(button.element.tagName).toBe('BUTTON')
  })

  it('holds whatever it is handed', () => {
    const other = defineComponent({ setup: () => () => h('span', 'said') })
    const divider = mount(Divider, { slots: { default: h(other) } })
    expect(divider.get('span').text()).toBe('said')
  })
})
