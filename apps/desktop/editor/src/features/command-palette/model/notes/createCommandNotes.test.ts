import { describe, expect, it, vi } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createCommandNotes } from './createCommandNotes'

describe('the open files a command can access', () => {
  it('include the files of the note editors and the files of the tab kinds', async () => {
    const flushTab = vi.fn(() => Promise.resolve())
    const onCloseCalled = vi.fn()
    const requestClose = vi.fn()
    const notes = createCommandNotes(
      {
        getTabAt: (p: string) => (p === 'Reached.md' ? 'reached:1' : null),
        getPath: (id: string) => (id === 'reached:1' ? 'Reached.md' : id),
        asking: (id: string) => id === 'asking:1',
        flushPending: vi.fn(() => Promise.resolve()),
        close: onCloseCalled,
      } as never,
      {
        tabs: {
          tabs: { value: [{ id: 'tab:1', title: 'One' }] },
          getTab: (id: string) =>
            id === 'tab:1'
              ? {
                  state: {},
                  kind: {
                    address: { getOpenTab: () => ({ path: 'Tab.md' }) },
                    events: { isAsking: () => true, flushPending: flushTab },
                  },
                }
              : null,
        },
        panes: { requestClose },
      } as never,
    )

    expect(notes.getTabAt('Reached.md')).toBe('reached:1')
    expect(notes.getTabAt('Tab.md')).toBe('tab:1')
    expect(notes.getPath('reached:1')).toBe('Reached.md')
    expect(notes.getPath('tab:1')).toBe('Tab.md')
    expect(notes.asking('asking:1')).toBe(true)
    expect(notes.asking('tab:1')).toBe(true)
    await notes.flushPending('tab:1')
    expect(flushTab).toHaveBeenCalledOnce()
    notes.close('tab:1')
    expect(onCloseCalled).toHaveBeenCalledWith('tab:1')
    expect(requestClose).toHaveBeenCalledWith('tab:1')
  })
})

describe('the tab of a file a command removes', () => {
  const FILE = 'cards/Shore birds.md'

  const tool: AnyTabKind = {
    kind: 'graph',
    open: () => ({}),
    pane: {},
    getTitle: () => 'Graph',
    display: { isTool: true },
    address: { getOpenTab: () => ({ path: FILE }) },
  }

  const deck: AnyTabKind = {
    kind: 'deck',
    open: () => ({}),
    pane: {},
    getTitle: () => 'Shore birds',
    address: { getOpenTab: () => ({ path: FILE }) },
    events: { flushPending: () => Promise.resolve(), onClose: () => true },
  }

  const reached = {
    getTabAt: () => null,
    getPath: (id: string) => id,
    asking: () => false,
    flushPending: () => Promise.resolve(),
    close: () => undefined,
    openFile: () => undefined,
  }

  it('is the tab that holds the file, not a tool standing on the same path', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([tool, deck])
    const toolId = await windowTabs.tabs.openTabOfKind('graph')
    const deckId = await windowTabs.tabs.openTabOfKind('deck', FILE)
    const notes = createCommandNotes(reached, windowTabs)

    notes.close(notes.getTabAt(FILE) ?? '')

    expect(windowTabs.tabs.getTab(deckId)).toBeNull()
    expect(windowTabs.tabs.getTab(toolId)).not.toBeNull()
  })
})
