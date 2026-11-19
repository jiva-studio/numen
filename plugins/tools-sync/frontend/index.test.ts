import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, SYNC_VIEW } from './index'
import { SYNC_LICENSE_ID } from './model/sync'
import { WORDS } from './words'

const setup = () => {
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context)
  const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
  return { context, wrapper }
}

describe('the sync plugin frontend', () => {
  it('registers the sync status view', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([SYNC_VIEW])
  })

  it('transitions from syncing to synced when the license is active and hides when deactivated', async () => {
    vi.useFakeTimers()
    try {
      const { context, wrapper } = setup()

      expect(wrapper.find('.sync-badge').exists()).toBe(false)
      context.setLicense(SYNC_LICENSE_ID, true)
      await nextTick()
      expect(wrapper.find('.sync-badge__label').text()).toBe(WORDS.syncingLabel)

      vi.advanceTimersByTime(2000)
      await nextTick()
      expect(wrapper.find('.sync-badge__label').text()).toBe(WORDS.syncedLabel)

      context.setLicense(SYNC_LICENSE_ID, false)
      await nextTick()
      expect(wrapper.find('.sync-badge').exists()).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })

  it('stops listening when the plugin context is disposed', async () => {
    const { context, wrapper } = setup()

    context.dispose()
    context.setLicense(SYNC_LICENSE_ID, true)
    await nextTick()

    expect(wrapper.find('.sync-badge').exists()).toBe(false)
  })
})
