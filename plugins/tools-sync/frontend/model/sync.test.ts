import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { createSyncState, SYNC_LICENSE_ID } from './sync'

describe('createSyncState', () => {
  it('enters syncing when activated and transitions to synced after the sync window', () => {
    vi.useFakeTimers()
    try {
      const context = createMockPluginContext()
      const state = createSyncState(context.licenses, 2000)

      state.start()
      expect(state.phase.value).toBe('hidden')

      context.setLicense(SYNC_LICENSE_ID, true)
      expect(state.phase.value).toBe('syncing')

      vi.advanceTimersByTime(2000)
      expect(state.phase.value).toBe('synced')

      context.setLicense(SYNC_LICENSE_ID, false)
      expect(state.phase.value).toBe('hidden')

      state.stop()
    } finally {
      vi.useRealTimers()
    }
  })

  it('starts in syncing when the license is initially active and stops listening on stop()', () => {
    vi.useFakeTimers()
    try {
      const context = createMockPluginContext({
        initialLicenses: { [SYNC_LICENSE_ID]: true },
      })
      const state = createSyncState(context.licenses, 2000)

      state.start()
      expect(state.phase.value).toBe('syncing')

      vi.advanceTimersByTime(2000)
      expect(state.phase.value).toBe('synced')

      state.stop()
      context.setLicense(SYNC_LICENSE_ID, false)
      expect(state.phase.value).toBe('synced')
    } finally {
      vi.useRealTimers()
    }
  })
})
