/**
 * The settings tab is handed the declaration of the shell as the core answered it, and the window
 * follows a write of a setting of the shell: the theme, the mode and the scales are applied after
 * the file changes. Every value here is invented.
 */
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createWindowSettingsPart } from '@/testing/windowSettings'
import { THEME_MODE, THEME_ORIGIN } from '@numen/core-client'
import { WORDS } from '@/shared/ui/words'
import { installSettingsCore, type SettingsCore } from '@/testing/settingsWire'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
  vi.restoreAllMocks()
})

/** What the core answers about the appearance once the file names the theme Dusk at 1.4 and 1.2. */
const DUSK = {
  themes: [
    {
      name: 'plugin:md.example.themes/dusk',
      title: 'Dusk',
      origin: THEME_ORIGIN.PLUGIN,
      pluginId: 'md.example.themes',
      isPinned: false,
    },
  ],
  applied: 'plugin:md.example.themes/dusk',
  mode: THEME_MODE.DARK,
  interfaceScale: 1.4,
  textScale: 1.2,
  interfaceScaleBounds: { min: 0.8, max: 2 },
  textScaleBounds: { min: 0.8, max: 1.75 },
}

const createWindow = () => {
  const core = {
    getSettings: vi.fn(async () => ({
      written: '{"v":1}',
      path: '/vault/numen.json',
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
    updateSettings: vi.fn(async () => undefined),
  } as unknown as SettingsCore
  const listThemes = vi.fn(() => DUSK)
  restore = installSettingsCore(core, {
    theme: {
      listThemes,
      readTheme: () => ({ css: ':root { --numen-surface: #223; }' }),
    },
  })
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = {
    groups: { handle: { openTab: vi.fn(async () => ''), openTabAdjacent: vi.fn(async () => '') } },
  } as unknown as WindowTabs
  const settings = createWindowSettingsPart({
    rest: createSettingsStore(SETTINGS_WORDS, log.getWriter('configured')),
    words: WORDS,
    log,
    windowTabs,
    onSizeChanged: vi.fn(),
    getSettingsViews: () => new Map(),
    getPlatform: () => 'linux',
  })
  return { settings, listThemes }
}

describe('the settings of the shell in the settings tab', () => {
  it('are the ones the core declared, with the value in force it resolved', async () => {
    const { settings } = createWindow()

    await settings.rest.start()

    expect(settings.configured.state.installation.plugin.app?.value).toMatchObject([
      { key: 'appearance.interfaceScale', value: 1.4 },
    ])
  })
})

describe('the appearance of the window after a write of the settings of the shell', () => {
  it('shows the theme, the mode and the scales the file now holds', async () => {
    const { settings } = createWindow()

    await (settings.configured.state.installation.settings.write([
      { at: ['app', 'appearance', 'theme'], value: '"plugin:md.example.themes/dusk"' },
    ]) as unknown as Promise<void>)

    await vi.waitFor(() => {
      expect(settings.appearance.state.applied.value).toBe('plugin:md.example.themes/dusk')
      expect(settings.appearance.state.mode.value).toBe('dark')
      expect(settings.appearance.preview.display.sized.value).toStrictEqual({
        interfaceScale: 1.4,
        textScale: 1.2,
      })
      expect(document.head.querySelector('style[data-appearance="theme"]')?.textContent).toContain(
        '--numen-surface: #223',
      )
    })
    settings.close()
  })

  it('stays as it is after a write of the settings of a plugin', async () => {
    const { settings, listThemes } = createWindow()

    await (settings.configured.state.installation.settings.write([
      { at: ['md.example.themes', 'accent'], value: '"teal"' },
    ]) as unknown as Promise<void>)

    expect(listThemes).not.toHaveBeenCalled()
    settings.close()
  })
})
