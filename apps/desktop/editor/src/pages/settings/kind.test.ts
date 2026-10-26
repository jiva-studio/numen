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

/** A window, writing down what it was asked to open. */
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
  ({
    themes: ref([]),
    applied: ref('preset/Numen.css'),
    mode: ref('system'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: vi.fn(),
    isSyncing: ref(false),
    isNesting: ref(false),
    headings: ref(0),
    headingsBounds: ref({ least: 1, most: 12 }),
    chooseHeadings: vi.fn(),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    choosePath: vi.fn(async () => ''),
    getSetting: () => undefined,
    getModels: () => [],
    write: vi.fn(),
    file: ref('/vaults/Physics/.numen/settings.json'),
    openFile: vi.fn(),
  }) satisfies Installation

describe('the settings tab', () => {
  it('holds the installation the window already keeps', () => {
    const target = installation()
    const settings = useSettingsTab(window_().handle, target)
    expect(settings.state.installation).toBe(target)
    expect(settings.kind.open('', 'tab')).toBe(settings.state)
  })

  it('is a tool that no window opens with', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.kind.isTool).toBe(true)
    expect(settings.kind).not.toHaveProperty('placement')
  })

  it('is called what the settings are called', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.kind.getTitle?.(settings.state)).toBe(words.settings)
  })

  // The settings are the installation's and not a file's, so every way to them
  // arrives at one tab.
  it('is one tab to a window, whatever it is opened on', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.kind.kind).toBe(SETTINGS)
    expect(settings.kind.identity?.('anything')).toBe(SETTINGS)
    expect(settings.kind.identity?.('something else')).toBe(SETTINGS)
  })

  it('is put in front when the window is requested to show it', () => {
    const { handle, opened } = window_()
    useSettingsTab(handle, installation()).openSettings()
    expect(opened).toEqual([SETTINGS])
  })

  it('is not open until a tab is opened or settings are requested for', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.isOpen.value).toBe(false)
  })

  it('is open once the window opens its tab', () => {
    const settings = useSettingsTab(window_().handle, installation())
    settings.kind.open('', 'tab')
    expect(settings.isOpen.value).toBe(true)
  })

  it('is open once the settings are requested for', () => {
    const settings = useSettingsTab(window_().handle, installation())
    settings.openSettings()
    expect(settings.isOpen.value).toBe(true)
  })
})
