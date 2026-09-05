/**
 * The settings tab: one to a window, holding the installation itself.
 */
import { describe, expect, it, vi } from 'vitest'
import { settling, type Installation } from './kind'
import { WORDS as words } from './words'
import { SETTINGS } from '../workspace'
import type { Host } from '../windowing'

/** A window, writing down what it was asked to open. */
const window_ = () => {
  const opened: string[] = []
  const host = {
    opens: async (kind: string) => {
      opened.push(kind)
      return kind
    },
  } as unknown as Host
  return { host, opened }
}

const installation = (): Installation =>
  ({
    themes: () => [],
    applied: () => 'preset/Numen.css',
    mode: () => 'system',
    pinned: () => false,
    sizes: () => ({ interfaceScale: 1, textScale: 1 }),
    bounds: () => ({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    chooses: vi.fn(),
    syncing: () => false,
    choosesSyncing: vi.fn(),
    hangs: () => false,
    parts: () => 0,
    choosesHanging: vi.fn(),
    choosesParts: vi.fn(),
    dayStarts: () => '04:00',
    latestDayStarts: () => '12:00',
    choosesDayStarts: vi.fn(),
    setting: () => undefined,
    models: () => [],
    writes: vi.fn(),
    file: () => '/vaults/Physics/.numen/settings.json',
    opensFile: vi.fn(),
  }) satisfies Installation

describe('the settings tab', () => {
  it('holds the installation the window already keeps', () => {
    const held = installation()
    const settings = settling(window_().host, held)
    expect(settings.held.installation).toBe(held)
    expect(settings.kind.opens('')).toBe(settings.held)
  })

  it('is called what the settings are called', () => {
    const settings = settling(window_().host, installation())
    expect(settings.kind.called(settings.held)).toBe(words.settings)
  })

  // The settings are the installation's and not a file's, so every way to them
  // arrives at one tab.
  it('is one tab to a window, whatever it is opened on', () => {
    const settings = settling(window_().host, installation())
    expect(settings.kind.kind).toBe(SETTINGS)
    expect(settings.kind.identity?.('anything')).toBe(SETTINGS)
    expect(settings.kind.identity?.('something else')).toBe(SETTINGS)
  })

  it('is put in front when the window is asked to show it', () => {
    const { host, opened } = window_()
    settling(host, installation()).shows()
    expect(opened).toEqual([SETTINGS])
  })
})
