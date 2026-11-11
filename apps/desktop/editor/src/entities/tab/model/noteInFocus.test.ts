import { describe, expect, it } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '../lib/kinds'
import { getNoteInFocus } from './noteInFocus'

const tool = (kind: string, getTarget?: AnyTabKind['address']): AnyTabKind => ({
  kind,
  open: (path: string) => ({ title: path, path }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  display: { isTool: true },
  ...(getTarget ? { address: getTarget } : {}),
})

const follow = (kind: string): AnyTabKind =>
  tool(kind, {
    navigate: () => undefined,
    getTarget: (state: { path: string }) => ({ path: state.path, title: `of ${state.path}` }),
  })

describe('the note a tool tab depends on', () => {
  it('is empty where no tool follows a note', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([tool('timeline')])
    await windowTabs.tabs.openTabOfKind('timeline', 'Tides.md')

    expect(getNoteInFocus(windowTabs)).toStrictEqual({ path: '', title: '' })
  })

  it('is the note of the tool tab opened last among those that follow a note', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([follow('graph'), follow('outline'), tool('timeline')])
    await windowTabs.tabs.openTabOfKind('graph', 'Tides.md')
    await windowTabs.tabs.openTabOfKind('outline', 'Moons.md')
    await windowTabs.tabs.openTabOfKind('timeline', 'Orbits.md')

    expect(getNoteInFocus(windowTabs)).toStrictEqual({ path: 'Moons.md', title: 'of Moons.md' })
  })
})
