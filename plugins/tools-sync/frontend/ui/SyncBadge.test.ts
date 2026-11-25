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
    expect(badge.find('.sync-badge__ring svg.sync-ring').exists()).toBe(true)
    expect(badge.find('.sync-badge__label').text()).toBe(WORDS.syncingLabel)
    expect(badge.attributes('aria-label')).toBe(WORDS.syncingDetail)
  })

  it('draws an arc of the share done on the ring and keeps it still', () => {
    const wrapper = mount(SyncBadge, { props: { phase: 'syncing', percent: 50 } })

    const ring = wrapper.get('.sync-badge__ring svg.sync-ring')
    const [arc, length] = (ring.get('.sync-ring__arc').attributes('stroke-dasharray') ?? '').split(
      ' ',
    )
    expect(Number(arc) / Number(length)).toBeCloseTo(0.5)
    expect(ring.classes()).not.toContain('sync-ring--indeterminate')
    expect(wrapper.get('.sync-badge__label').text()).toBe(WORDS.syncingPercent(50))
  })

  it('orbits a short arc while the total is unknown', () => {
    const wrapper = mount(SyncBadge, { props: { phase: 'syncing', percent: null } })

    const ring = wrapper.get('svg.sync-ring')
    const [arc, length] = (ring.get('.sync-ring__arc').attributes('stroke-dasharray') ?? '').split(
      ' ',
    )
    expect(Number(arc) / Number(length)).toBeCloseTo(0.25)
    expect(ring.classes()).toContain('sync-ring--indeterminate')
    expect(wrapper.get('.sync-badge__label').text()).toBe(WORDS.syncingLabel)
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
