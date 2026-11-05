/**
 * The settings tab is handed the declaration of the shell as the core answered it.
 */
import { describe, expect, it, vi } from 'vitest'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

describe('the settings of the shell in the settings tab', () => {
  it('are the ones the core declared, with the value in force it resolved', async () => {
    const core = {
      getSettings: vi.fn(async () => ({
        written: '{"v":1}',
        path: '/vault/numen.json',
        models: [],
        app: [
          {
            key: 'appearance.interfaceScale',
            type: 'range' as const,
            title: 'Interface size',
            description: '',
            default: 1,
            value: 1.4,
          },
        ],
      })),
    } as unknown as SettingsPort
    const log = { getWriter: () => vi.fn() } as unknown as MessageLog
    const windowTabs = {
      handle: { openTab: vi.fn(async () => ''), openTabAdjacent: vi.fn(async () => '') },
    } as unknown as ReturnType<typeof useWindowTabs>
    const settings = useSettings({ core, words: WORDS, log, windowTabs, onSizeChanged: vi.fn() })

    await settings.rest.start()

    expect(settings.configured.state.installation.app?.value).toMatchObject([
      { key: 'appearance.interfaceScale', value: 1.4 },
    ])
  })
})
