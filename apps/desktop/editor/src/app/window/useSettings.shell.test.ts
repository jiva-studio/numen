/**
 * The settings tab is handed the declaration of the shell as the core answered it, and the window
 * follows a write of a setting of the shell: the theme, the mode and the scales are applied after
 * the file changes. Every value here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useSettings } from './useSettings'
import { themes } from '@/entities/settings'
import { WORDS } from '@/shared/words'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'

afterEach(() => {
  vi.restoreAllMocks()
})

const createWindow = () => {
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
  vi.spyOn(settings.rest, 'start').mockResolvedValue()
  vi.spyOn(settings.rest, 'writeSettings').mockResolvedValue()
  return settings
}

/** What the core answers about the appearance once the file names the theme Dusk at 1.4 and 1.2. */
const DUSK = {
  themes: [
    {
      name: 'plugin:md.example.themes/dusk',
      title: 'Dusk',
      isBuiltIn: false,
      pluginId: 'md.example.themes',
      isPinned: false,
    },
  ],
  applied: 'plugin:md.example.themes/dusk',
  mode: 'dark' as const,
  sizes: { interfaceScale: 1.4, textScale: 1.2 },
  bounds: {
    interfaceScale: { min: 0.8, max: 2 },
    textScale: { min: 0.8, max: 1.75 },
  },
}

describe('the settings of the shell in the settings tab', () => {
  it('are the ones the core declared, with the value in force it resolved', async () => {
    const settings = createWindow()
    vi.mocked(settings.rest.start).mockRestore()

    await settings.rest.start()

    expect(settings.configured.state.installation.app?.value).toMatchObject([
      { key: 'appearance.interfaceScale', value: 1.4 },
    ])
  })
})

describe('the appearance of the window after a write of the settings of the shell', () => {
  it('shows the theme, the mode and the scales the file now holds', async () => {
    const settings = createWindow()
    vi.spyOn(themes, 'getAppearance').mockResolvedValue(DUSK)
    vi.spyOn(themes, 'readTheme').mockResolvedValue(':root { --numen-surface: #223; }')

    await (settings.configured.state.installation.write([
      { at: ['app', 'appearance', 'theme'], value: '"plugin:md.example.themes/dusk"' },
    ]) as unknown as Promise<void>)

    await vi.waitFor(() => {
      expect(settings.appearance.applied.value).toBe('plugin:md.example.themes/dusk')
      expect(settings.appearance.mode.value).toBe('dark')
      expect(settings.appearance.sized.value).toStrictEqual({ interfaceScale: 1.4, textScale: 1.2 })
      expect(document.head.querySelector('style[data-appearance="theme"]')?.textContent).toContain(
        '--numen-surface: #223',
      )
    })
    settings.close()
  })

  it('stays as it is after a write of the settings of a plugin', async () => {
    const settings = createWindow()
    const read = vi.spyOn(themes, 'getAppearance').mockResolvedValue(DUSK)

    await (settings.configured.state.installation.write([
      { at: ['md.example.themes', 'accent'], value: '"teal"' },
    ]) as unknown as Promise<void>)

    expect(read).not.toHaveBeenCalled()
    settings.close()
  })
})
