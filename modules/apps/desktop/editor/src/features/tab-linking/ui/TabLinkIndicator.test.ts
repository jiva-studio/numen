import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TabLinkIndicator from './TabLinkIndicator.vue'

describe('TabLinkIndicator', () => {
  it('does not render when isLinked is false', () => {
    const wrapper = mount(TabLinkIndicator, {
      props: { isLinked: false },
    })
    expect(wrapper.find('.tab-link-indicator').exists()).toBe(false)
  })

  it('renders link icon when isLinked is true with default color', () => {
    const wrapper = mount(TabLinkIndicator, {
      props: { isLinked: true },
    })
    expect(wrapper.find('.tab-link-indicator').exists()).toBe(true)
    const icon = wrapper.find('.tab-link-indicator__icon')
    expect(icon.exists()).toBe(true)
  })

  it('renders with correct color for all colorIndex values', () => {
    const colors: Record<number, string> = {
      1: 'var(--numen-blue, #3b82f6)',
      2: 'var(--numen-leaf, #22c55e)',
      3: 'var(--numen-amber, #f59e0b)',
      4: 'var(--numen-purple, #a855f7)',
      99: 'var(--numen-blue, #3b82f6)',
    }

    for (const [colorIndexStr, expectedColor] of Object.entries(colors)) {
      const colorIndex = Number(colorIndexStr)
      const wrapper = mount(TabLinkIndicator, {
        props: { isLinked: true, colorIndex },
      })
      const icon = wrapper.find('.tab-link-indicator__icon')
      expect(icon.attributes('style')).toContain(`color: ${expectedColor}`)
    }
  })
})
