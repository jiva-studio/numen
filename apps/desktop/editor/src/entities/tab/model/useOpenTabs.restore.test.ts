/**
 * The active tab the core keeps when a tool in its own pane is in focus: the tab the person worked
 * on is the one that is shown again after a restart.
 */
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { getPanes } from '@numen/workbench'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '../lib/kinds'
import { parseSavedTabs, useOpenTabs } from './useOpenTabs'

const createKind = (kind: string, display?: AnyTabKind['display']): AnyTabKind => ({
  kind,
  open: (path: string) => ({ path }),
  pane: {},
  getTitle: (state: { path: string }) => state.path || kind,
  ...(display ? { display } : {}),
  address: { getOpenTab: (state: { path: string }) => ({ path: state.path }) },
})

const createWindow = () => {
  const window = createTestTabs()
  window.groups.registerKind(createKind('reader'))
  window.groups.registerKind(createKind('note'))
  window.groups.registerKind(
    createKind('files', { isTool: true, placement: { pane: 'aside', order: 1 } }),
  )
  return window
}

const createOpenTabs = (window: ReturnType<typeof createWindow>) =>
  useOpenTabs({
    windowTabs: window,
    isWriting: ref(false),
    writeMessage: () => undefined,
    writeOpenTabs: () => Promise.resolve(),
  })

describe('the open tabs of a window whose files tab is in focus', () => {
  it('restores the adjacent note as the active tab after a restart', () => {
    const window = createWindow()
    window.restored.restoreTabs(
      [
        { id: 'files-1', kind: 'files', path: '' },
        { id: 'reader-1', kind: 'reader', path: 'assets/shore.pdf' },
        { id: 'note-1', kind: 'note', path: 'notes/Entropy.md' },
      ],
      'note-1',
    )
    window.panes.show('files-1')
    expect(window.groups.handle.front()?.id).toBe('files-1')

    const { open } = createOpenTabs(window)

    expect(open.value.activeId).toBe('note-1')
    const restarted = createWindow()
    restarted.restored.restoreTabs(parseSavedTabs(open.value), open.value.activeId)
    expect(restarted.groups.handle.front()?.id).toBe('note-1')
  })

  it('keep the files tab as the active tab where no other tab shows', () => {
    const window = createWindow()
    window.restored.restoreTabs([{ id: 'files-1', kind: 'files', path: '' }], 'files-1')

    expect(createOpenTabs(window).open.value.activeId).toBe('files-1')
  })
})

describe('the tab in front of each pane after a restart', () => {
  it('is the tab that was shown last in the pane', () => {
    const create = () => {
      const window = createWindow()
      window.groups.registerKind(
        createKind('plex', { isTool: true, placement: { pane: 'main', order: 0 } }),
      )
      return window
    }
    const window = create()
    window.restored.restoreTabs(
      [
        { id: 'reader-1', kind: 'reader', path: 'assets/shore.pdf' },
        { id: 'plex-1', kind: 'plex', path: 'notes/Entropy.md' },
        { id: 'files-1', kind: 'files', path: '' },
      ],
      'files-1',
    )
    window.panes.show('reader-1')
    window.panes.show('plex-1')
    window.panes.show('files-1')

    const { open } = createOpenTabs(window)
    const restarted = create()
    restarted.restored.restoreTabs(parseSavedTabs(open.value), open.value.activeId)

    const fronts = getPanes(restarted.panes.layout.value.root).map((one) => one.active)
    expect(fronts).toEqual(['plex-1', 'files-1'])
  })
})
