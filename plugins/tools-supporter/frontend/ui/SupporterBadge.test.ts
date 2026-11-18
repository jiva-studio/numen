import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SupporterBadge from './SupporterBadge.vue'
import { WORDS } from '../words'

describe('SupporterBadge', () => {
  it('renders the supporter badge and thank-you tooltip when active', () => {
    const wrapper = mount(SupporterBadge, {
      props: { isActive: true },
    })

    const badge = wrapper.find('.supporter-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.find('.supporter-badge__label').text()).toBe(WORDS.badgeLabel)
    expect(badge.attributes('title')).toBe(WORDS.thankYou)
    expect(badge.attributes('aria-label')).toBe(WORDS.thankYou)
  })

  it('renders nothing when inactive', () => {
    const wrapper = mount(SupporterBadge, {
      props: { isActive: false },
    })

    expect(wrapper.find('.supporter-badge').exists()).toBe(false)
  })
})
