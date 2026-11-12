/**
 * The settings tab: one to a window, holding the installation itself.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useSettingsTab } from './kind'
import type { Installation } from './types'
import { WORDS as words } from './words'
import { SETTINGS } from '@/entities/tab'
import type { WindowHandle } from '@/entities/tab'
import { createInstallation } from '@/pages/settings/testing/installation'

const window_ = () => {
  const openedTab: string[] = []
  const handle = {
    openTab: async (kind: string) => {
      openedTab.push(kind)
      return kind
    },
  } as unknown as WindowHandle
  return { handle, opened: openedTab }
}

const installation = (): Installation =>
  createInstallation({
    write: vi.fn(),
    file: ref('/vaults/Physics/.numen/settings.json'),
  })

describe('the settings tab', () => {
  it('holds the installation the window already keeps', () => {
    const target = installation()
    const settings = useSettingsTab(window_().handle, target, vi.fn())
    expect(settings.state.installation).toBe(target)
    expect(settings.kind.open('', 'tab')).toBe(settings.state)
  })

  it('is a tool that no window opens with', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    expect(settings.kind.display?.isTool).toBe(true)
    expect(settings.kind).not.toHaveProperty('placement')
  })

  it('is called what the settings are called', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    expect(settings.kind.getTitle?.(settings.state)).toBe(words.settings)
  })

  // The settings are the installation's and not a file's, so every way to them
  // arrives at one tab.
  it('is one tab to a window, whatever it is opened on', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    expect(settings.kind.kind).toBe(SETTINGS)
    expect(settings.kind.address?.identity?.('anything')).toBe(SETTINGS)
    expect(settings.kind.address?.identity?.('something else')).toBe(SETTINGS)
  })

  it('is put in front when the window is requested to show it', () => {
    const { handle, opened } = window_()
    useSettingsTab(handle, installation(), vi.fn()).openSettings()
    expect(opened).toEqual([SETTINGS])
  })

  it('is not open until a tab is opened or settings are requested for', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    expect(settings.isOpen.value).toBe(false)
  })

  it('is open once the window opens its tab', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    settings.kind.open('', 'tab')
    expect(settings.isOpen.value).toBe(true)
  })

  it('is open once the settings are requested for, tracking pluginId and filter', () => {
    const settings = useSettingsTab(window_().handle, installation(), vi.fn())
    expect(settings.state.initialPluginId?.value).toBeNull()
    expect(settings.state.initialSearchQuery?.value).toBe('')

    settings.kind.open('plugin-a', 'tab')
    expect(settings.state.initialPluginId?.value).toBe('plugin-a')
    settings.kind.open(undefined as never, 'tab')
    expect(settings.state.initialPluginId?.value).toBe('plugin-a')

    settings.openSettings('plugin-b')
    expect(settings.isOpen.value).toBe(true)
    expect(settings.state.initialPluginId?.value).toBe('plugin-b')
    expect(settings.state.initialSearchQuery?.value).toBe('')

    settings.openSettings('plugin-c', 'theme')
    expect(settings.state.initialPluginId?.value).toBeNull()
    expect(settings.state.initialSearchQuery?.value).toBe('theme')

    settings.state.initialPluginId!.value = 'kept'
    settings.openSettings()
    expect(settings.state.initialPluginId?.value).toBe('kept')
    expect(settings.state.initialSearchQuery?.value).toBe('theme')
  })
})
