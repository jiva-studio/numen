/**
 * The active tab the core keeps when a tool in its own pane is in focus: the tab the person worked
 * on is the one that is shown again after a restart.
 */
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { branch, getPanes, pane, type Workspace } from '@numen/workbench'
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
  window.kinds.registerKind(createKind('reader'))
  window.kinds.registerKind(createKind('note'))
  window.kinds.registerKind(
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
    expect(window.handle.getFrontTab()?.id).toBe('files-1')

    const { open } = createOpenTabs(window)

    expect(open.value.activeId).toBe('note-1')
    const restarted = createWindow()
    restarted.restored.restoreTabs(parseSavedTabs(open.value), open.value.activeId)
    expect(restarted.handle.getFrontTab()?.id).toBe('note-1')
  })

  it('keep the files tab as the active tab where no other tab shows', () => {
    const window = createWindow()
    window.restored.restoreTabs([{ id: 'files-1', kind: 'files', path: '' }], 'files-1')

    expect(createOpenTabs(window).open.value.activeId).toBe('files-1')
  })

  it('picks the most recently shown non-tool tab that does not accept text across panes', async () => {
    const window = createWindow()
    window.kinds.registerKind({
      ...createKind('chat', { placement: { pane: 'aside', order: 2 } }),
      events: { receiveText: () => undefined },
    })
    const first = await window.tabs.openTabOfKind('reader', 'assets/shore.pdf')
    const second = await window.handle.openTabAdjacent('note', 'notes/Entropy.md')
    const hidden = await window.tabs.openTabOfKind('note', 'notes/Hidden.md')
    window.panes.show(second)
    const chat = await window.handle.openTabAdjacent('chat', '')
    const files = await window.handle.openTabAdjacent('files', '')
    window.panes.show(files)

    expect(first).toBeDefined()
    expect(hidden).toBeDefined()
    expect(chat).toBeDefined()
    expect(createOpenTabs(window).open.value.activeId).toBe(second)
  })
})

describe('the tab in front of each pane after a restart', () => {
  it('is the tab that was shown last in the pane', () => {
    const create = () => {
      const window = createWindow()
      window.kinds.registerKind(
        createKind('plex', { isTool: true, placement: { pane: 'main', order: 0 } }),
      )
      return window
    }
    const window = create()
    window.restored.restoreTabs(
      [
        { id: 'note-1', kind: 'note', path: 'notes/First.md' },
        { id: 'reader-1', kind: 'reader', path: 'assets/shore.pdf' },
        { id: 'plex-1', kind: 'plex', path: 'notes/Entropy.md' },
        { id: 'files-1', kind: 'files', path: 'a' },
        { id: 'files-2', kind: 'files', path: 'b' },
        { id: 'files-3', kind: 'files', path: 'c' },
      ],
      'files-3',
    )
    window.panes.show('note-1')
    window.panes.show('reader-1')
    window.panes.show('plex-1')
    window.panes.show('files-1')
    window.panes.show('files-2')
    window.panes.show('files-3')

    const { open } = createOpenTabs(window)
    const restarted = create()
    restarted.restored.restoreTabs(parseSavedTabs(open.value), open.value.activeId)

    const fronts = getPanes(restarted.panes.layout.value.root).map((one) => one.active)
    expect(fronts).toEqual(['plex-1', 'files-3'])
  })
})

describe('saved pane layouts', () => {
  it('restores nested splits, tab order, pane selections and focus independently of the reported active tab', () => {
    const original = createWindow()
    original.restored.restoreTabs(
      [
        { id: 'note', kind: 'note', path: 'Notes/First.md' },
        { id: 'reader', kind: 'reader', path: 'File.pdf' },
        { id: 'files', kind: 'files', path: '' },
        { id: 'missing', kind: 'missing.plugin', path: 'Other.md' },
      ],
      'note',
    )
    const layout: Workspace = {
      axis: 'vertical',
      focus: 'file-pane',
      root: branch(
        'root',
        [
          pane('stack', ['reader', 'note'], 'reader'),
          branch('lower', [pane('file-pane', ['files']), pane('last', ['missing'])], [0.2, 0.8]),
        ],
        [0.45, 0.55],
      ),
    }
    original.panes.layout.value = layout
    const snapshot = createOpenTabs(original).open.value
    expect(snapshot.activeId).not.toBe('files')

    const restored = createWindow()
    restored.restored.restoreTabs(parseSavedTabs(snapshot), snapshot.activeId, snapshot.layout)

    expect(restored.panes.layout.value).toStrictEqual(layout)
    expect(restored.handle.getFrontTab()?.id).toBe('files')
    expect(restored.tabs.getTab('missing')?.kind.kind).toBe('missing.plugin')
  })

  it('restores saved tabs by kind placement when their layout has an obsolete tab id', () => {
    const window = createWindow()
    const layout = { axis: 'vertical', root: pane('stack', ['gone']), focus: 'stack' }
    window.restored.restoreTabs(
      [
        { id: 'note', kind: 'note', path: 'Notes/First.md' },
        { id: 'files', kind: 'files', path: '' },
      ],
      'note',
      JSON.stringify(layout),
    )

    expect(getPanes(window.panes.layout.value.root).map((p) => p.tabs)).toEqual([
      ['note'],
      ['files'],
    ])
    expect(window.handle.getFrontTab()?.id).toBe('note')
  })
})
