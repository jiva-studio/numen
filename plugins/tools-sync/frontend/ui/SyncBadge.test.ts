import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SyncBadge from './SyncBadge.vue'
import { WORDS } from '../words'

describe('SyncBadge', () => {
  it('pings the dot and names the state while synchronizing', () => {
    const wrapper = mount(SyncBadge, { props: { phase: 'syncing' } })

    const badge = wrapper.get('.sync-badge')
    expect(badge.get('.sync-badge__dot').classes()).toContain('sync-badge__dot--syncing')
    expect(badge.find('svg').exists()).toBe(false)
    expect(badge.get('.sync-badge__label').text()).toBe(WORDS.syncingLabel)
    expect(badge.attributes('aria-label')).toBe(WORDS.syncingDetail)
  })

  it('writes the percent next to the pinging dot', () => {
    const wrapper = mount(SyncBadge, { props: { phase: 'syncing', percent: 50 } })

    expect(wrapper.get('.sync-badge__label').text()).toBe(WORDS.syncingPercent(50))
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('keeps the dot still when synced and when waiting', () => {
    for (const phase of ['synced', 'waiting'] as const) {
      const wrapper = mount(SyncBadge, { props: { phase } })

      expect(wrapper.get('.sync-badge__dot').classes()).not.toContain('sync-badge__dot--syncing')
    }
  })

  it('renders the synced indicator once synchronization completes', () => {
    const wrapper = mount(SyncBadge, {
      props: { phase: 'synced' },
    })

    const badge = wrapper.find('.sync-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.find('.sync-badge__dot').exists()).toBe(true)
    expect(badge.find('.sync-badge__label').text()).toBe(WORDS.syncedLabel)
    expect(badge.attributes('aria-label')).toBe(WORDS.syncedDetail)
  })

  it('renders nothing when hidden', () => {
    const wrapper = mount(SyncBadge, {
      props: { phase: 'hidden' },
    })

    expect(wrapper.find('.sync-badge').exists()).toBe(false)
  })
})
