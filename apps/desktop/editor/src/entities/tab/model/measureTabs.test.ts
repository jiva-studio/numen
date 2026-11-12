import { describe, expect, it, vi } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '../lib/kinds'
import { measureTabs } from './measureTabs'

const kind = (name: string, measure?: () => void): AnyTabKind => ({
  kind: name,
  open: (path: string) => ({ title: path, ...(measure ? { measure } : {}) }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
})

describe('measuring the open tabs', () => {
  it('lays out again the tabs whose state can measure, and leaves the others', async () => {
    const measureNote = vi.fn()
    const measureReader = vi.fn()
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([
      kind('note', measureNote),
      kind('reader', measureReader),
      kind('graph'),
    ])
    await windowTabs.tabs.openTabOfKind('note', 'Tides.md')
    await windowTabs.tabs.openTabOfKind('reader', 'Moons.pdf')
    await windowTabs.tabs.openTabOfKind('graph', 'Orbits.md')

    measureTabs(windowTabs)

    expect(measureNote).toHaveBeenCalledOnce()
    expect(measureReader).toHaveBeenCalledOnce()
  })
})
