import { describe, expect, it, vi } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createCommandNotes } from './createCommandNotes'

describe('the open files a command can access', () => {
  it('include the files of the note editors and the files of the tab kinds', async () => {
    const flushTab = vi.fn(() => Promise.resolve())
    const onCloseCalled = vi.fn()
    const closeTab = vi.fn()
    const openFile = vi.fn()
    const tabMap: Record<string, object | null> = {
      'tab:missing': null,
      'tab:bare': { state: {}, kind: {} },
      'tab:target': {
        state: {},
        kind: {
          address: { getTarget: () => ({ file: 'Target.pdf', path: '' }) },
          events: {},
        },
      },
      'tab:1': {
        state: {},
        kind: {
          address: { getOpenTab: () => ({ path: 'Tab.md' }) },
          events: { isAsking: () => true, flushPending: flushTab },
        },
      },
    }
    const notes = createCommandNotes(
      {
        openFile,
        getTabAt: (p: string) => (p === 'Reached.md' ? 'reached:1' : null),
        getPath: (id: string) => (id === 'reached:1' ? 'Reached.md' : id),
        isAsking: (id: string) => id === 'asking:1',
        flushPending: vi.fn(() => Promise.resolve()),
        close: onCloseCalled,
      } as never,
      {
        tabs: {
          tabs: {
            value: [
              { id: 'tab:missing', title: 'Missing' },
              { id: 'tab:bare', title: 'Bare' },
              { id: 'tab:target', title: 'Target' },
              { id: 'tab:1', title: 'One' },
            ],
          },
          getTab: (id: string) => tabMap[id] ?? null,
        },
        panes: { closeTab },
      } as never,
    )

    notes.openFile('Tab.md', 'Tab', 'split', { line: 3 })
    expect(openFile).toHaveBeenCalledWith('Tab.md', 'Tab', 'split', { line: 3 })
    expect(notes.getTabAt('')).toBeNull()
    expect(notes.getTabAt('Reached.md')).toBe('reached:1')
    expect(notes.getTabAt('Target.pdf')).toBe('tab:target')
    expect(notes.getTabAt('Tab.md')).toBe('tab:1')
    expect(notes.getPath('reached:1')).toBe('Reached.md')
    expect(notes.getPath('tab:1')).toBe('Tab.md')
    expect(notes.getPath('tab:missing')).toBe('tab:missing')
    expect(notes.getPath('tab:bare')).toBe('tab:bare')
    expect(notes.getPath('tab:target')).toBe('tab:target')
    expect(notes.isAsking('asking:1')).toBe(true)
    expect(notes.isAsking('tab:1')).toBe(true)
    expect(notes.isAsking('tab:missing')).toBe(false)
    expect(notes.isAsking('tab:bare')).toBe(false)
    expect(notes.isAsking('tab:target')).toBe(false)
    await notes.flushPending('tab:missing')
    await notes.flushPending('tab:bare')
    await notes.flushPending('tab:target')
    await notes.flushPending('tab:1')
    expect(flushTab).toHaveBeenCalledOnce()
    notes.close('tab:1')
    expect(onCloseCalled).toHaveBeenCalledWith('tab:1')
    expect(closeTab).toHaveBeenCalledWith('tab:1')
  })
})

describe('the tab to close when a file is removed', () => {
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
    isAsking: () => false,
    flushPending: () => Promise.resolve(),
    close: () => undefined,
    openFile: () => undefined,
  }

  it('is the tab of the file, not a tool tab with the same path', async () => {
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([tool, deck])
    const toolId = await windowTabs.tabs.openTabOfKind('graph')
    const deckId = await windowTabs.tabs.openTabOfKind('deck', FILE)
    const notes = createCommandNotes(reached, windowTabs)

    notes.close(notes.getTabAt(FILE) ?? '')

    expect(windowTabs.tabs.getTab(deckId)).toBeNull()
    expect(windowTabs.tabs.getTab(toolId)).not.toBeNull()
  })
})
