import { describe, expect, it, vi } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import { mockVaultState } from '@/testing/answers'
import { getTabs, mountWindow } from '@/testing/window'

describe('restoring a session with a disabled plugin', () => {
  it('restores a session holding a tab of a disabled plugin without throwing', async () => {
    try {
      mockVaultState.detachedTab = {
        id: 'disabled-plugin:/docs/sample.pdf',
        kind: 'disabled-plugin',
        path: '/docs/sample.pdf',
        title: 'sample.pdf',
        state: '',
      }
      vi.stubGlobal(
        'location',
        new URL('http://localhost/?window=secondary&id=disabled-plugin:/docs/sample.pdf'),
      )

      const window = await mountWindow()
      expect(getTabs(window)).toEqual([
        { id: 'disabled-plugin:/docs/sample.pdf', title: 'sample.pdf' },
      ])
      expect(window.text()).toContain('Plugin unavailable')
      expect(window.text()).toContain('The plugin for this tab is disabled or missing.')
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('restores a disabled plugin tab in windowTabs directly without throwing', async () => {
    const window = createTestTabs()
    const id = await window.tabs.createTab('disabled-plugin', '/docs/sample.pdf')

    expect(id).toBe('disabled-plugin:/docs/sample.pdf')
    expect(window.tabs.tabs.value).toEqual([{ id, title: 'sample.pdf' }])
  })
})
