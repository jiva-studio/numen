/**
 * The ranges a reader tab highlights: the reader tab kind opens a tab with none, and a file
 * opened at ranges puts a new array of them on the state of its tab.
 */
import { describe, expect, it, vi } from 'vitest'
import type { TextRange } from '@numen/plugin-sdk'
import { createReaderTabKind } from './customTabKind'
import { createPluginHost } from './host'
import type { SourceReader } from './tabKinds'
import type { PluginHostDeps } from './types'

describe('the ranges of a reader tab', () => {
  it('are none on a tab just opened', () => {
    const kind = createReaderTabKind('sample.plugin', { id: 'reader:bin', component: null })

    const state = kind.open('library/Birds.bin', 'tab-1') as { ranges?: unknown }

    expect(state.ranges).toBeNull()
  })

  it('are set as a new array on the tab of a file opened at them', async () => {
    const states: Record<string, { ranges?: unknown; selector?: unknown }> = { 'tab-1': {} }
    let reader: SourceReader | undefined
    const openTab = vi.fn(async () => 'tab-1')
    const deps: PluginHostDeps = {
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader: vi.fn((_ext: string, read: SourceReader) => {
        reader = read
        return vi.fn()
      }),
      openTab,
      getTabState: (id: string) => states[id],
      getReaderExtensions: () => ['bin'],
    }
    createPluginHost(deps)
      .createContext('sample.plugin')
      .registerReader({ id: 'reader:bin', component: null })
    const ranges: readonly TextRange[] = [
      { start: 1500, end: 1580 },
      { start: 400, end: 1100 },
    ]

    reader?.('library/Birds.bin', null, ranges)
    await vi.waitFor(() => expect(states['tab-1']?.ranges).toStrictEqual(ranges))
    const first = states['tab-1']?.ranges
    reader?.('library/Birds.bin', null, ranges)
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(2))
    await vi.waitFor(() => expect(states['tab-1']?.ranges).not.toBe(first))

    expect(openTab.mock.calls[0]).toStrictEqual(['reader:bin', 'library/Birds.bin'])
    expect(states['tab-1']?.ranges).not.toBe(ranges)
    expect(states['tab-1']?.ranges).toStrictEqual(ranges)
    expect(states['tab-1']?.selector).toBeUndefined()
  })
})
