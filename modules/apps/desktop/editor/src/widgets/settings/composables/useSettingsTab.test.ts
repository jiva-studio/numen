/**
 * The settings tab: one to a window, holding the installation itself.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useSettingsTab, type Installation } from './useSettingsTab'
import { WORDS as words } from '../words'
import { SETTINGS } from '@/entities/tab/workspace'
import type { WindowHandle } from '@/entities/tab/windowTabs'

/** A window, writing down what it was asked to open. */
const window_ = () => {
  const opened: string[] = []
  const handle = {
    opens: async (kind: string) => {
      opened.push(kind)
      return kind
    },
  } as unknown as WindowHandle
  return { handle, opened }
}

const installation = (): Installation =>
  ({
    themes: ref([]),
    applied: ref('preset/Numen.css'),
    mode: ref('system'),
    pinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    chooses: vi.fn(),
    syncing: ref(false),
    hangs: ref(false),
    parts: ref(0),
    partsBounds: ref({ least: 1, most: 12 }),
    choosesParts: vi.fn(),
    dayStarts: ref('04:00'),
    latestDayStarts: ref('12:00'),
    choosesDayStarts: vi.fn(),
    setting: () => undefined,
    models: () => [],
    writes: vi.fn(),
    file: ref('/vaults/Physics/.numen/settings.json'),
    opensFile: vi.fn(),
  }) satisfies Installation

describe('the settings tab', () => {
  it('holds the installation the window already keeps', () => {
    const held = installation()
    const settings = useSettingsTab(window_().handle, held)
    expect(settings.state.installation).toBe(held)
    expect(settings.kind.opens('')).toBe(settings.state)
  })

  it('is called what the settings are called', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.kind.called?.(settings.state)).toBe(words.settings)
  })

  // The settings are the installation's and not a file's, so every way to them
  // arrives at one tab.
  it('is one tab to a window, whatever it is opened on', () => {
    const settings = useSettingsTab(window_().handle, installation())
    expect(settings.kind.kind).toBe(SETTINGS)
    expect(settings.kind.identity?.('anything')).toBe(SETTINGS)
    expect(settings.kind.identity?.('something else')).toBe(SETTINGS)
  })

  it('is put in front when the window is asked to show it', () => {
    const { handle, opened } = window_()
    useSettingsTab(handle, installation()).shows()
    expect(opened).toEqual([SETTINGS])
  })
})
