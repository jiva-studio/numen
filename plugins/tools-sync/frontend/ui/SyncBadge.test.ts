import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SyncBadge from './SyncBadge.vue'
import { WORDS } from '../words'

describe('SyncBadge', () => {
  it('renders the syncing spinner and label while synchronizing', () => {
    const wrapper = mount(SyncBadge, {
      props: { phase: 'syncing' },
    })

    const badge = wrapper.find('.sync-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.find('.sync-badge__spinner').exists()).toBe(true)
    expect(badge.find('.sync-badge__label').text()).toBe(WORDS.syncingLabel)
    expect(badge.attributes('aria-label')).toBe(WORDS.syncingDetail)
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
