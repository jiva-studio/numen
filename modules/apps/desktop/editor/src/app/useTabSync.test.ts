import { describe, expect, it, vi } from 'vitest'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from './useTabSync'

describe('useTabSync', () => {
  it('synchronizes path to linked note, plex, and files tabs', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('plex-1', 'note-1')
    tabLinks.linkTabs('plex-1', 'files-1')

    const loadInTabMock = vi.fn()
    const revealPathMock = vi.fn()
    const plexGoMock = vi.fn()

    const heldMock: any = {
      getTab: (id: string) => {
        if (id === 'note-1') return { kind: { kind: 'note' }, state: { id: 'note-state-1' } }
        if (id === 'plex-1') return { kind: { kind: 'plex' } }
        if (id === 'files-1') return { kind: { kind: 'files' } }
        return null
      },
      getTabStateIn: (id: string, kind: string) => {
        if (id === 'plex-1' && kind === 'plex') {
          return { view: { here: { value: 'old.md' }, go: plexGoMock } }
        }
        if (id === 'files-1' && kind === 'files') {
          return { list: { revealPath: revealPathMock } }
        }
        return null
      },
    }

    const editingMock: any = {
      noted: { loadInTab: loadInTabMock },
    }

    const sync = useTabSync({ held: heldMock, tabLinks, editing: editingMock })

    // Sync from Plex -> Note & Files
    const synced = sync.syncPathFromTab('plex-1', 'test.md', 'Test')
    expect(synced).toBe(true)
    expect(loadInTabMock).toHaveBeenCalledWith('note-state-1', 'test.md', 'Test')
    expect(revealPathMock).toHaveBeenCalledWith('test.md')

    // Sync from Files -> Plex & Note
    const syncedFromFiles = sync.syncPathFromTab('files-1', 'other.md')
    expect(syncedFromFiles).toBe(true)
    expect(loadInTabMock).toHaveBeenCalledWith('note-state-1', 'other.md', '')
    expect(plexGoMock).toHaveBeenCalledWith('other.md')
  })

  it('handles edge cases in tab sync safely', () => {
    const tabLinks = useTabLinks()
    const loadInTabMock = vi.fn()
    const heldMock: any = {
      getTab: (id: string) => {
        if (id === 'missing-state') return { kind: { kind: 'plex' } }
        if (id === 'agent-tab') return { kind: { kind: 'agent' } }
        if (id === 'already-at-path') return { kind: { kind: 'plex' } }
        return null
      },
      getTabStateIn: (id: string) => {
        if (id === 'already-at-path') {
          return { view: { here: { value: 'same.md' }, go: vi.fn() } }
        }
        return null
      },
    }
    const sync = useTabSync({
      held: heldMock,
      tabLinks,
      editing: { noted: { loadInTab: loadInTabMock } } as any,
    })

    expect(sync.syncPathFromTab('', 'test.md')).toBe(false)
    expect(sync.syncPathFromTab('tab-1', '')).toBe(false)
    expect(sync.syncPathFromTab('unlinked-tab', 'test.md')).toBe(false)

    tabLinks.linkTabs('src-tab', 'nonexistent-tab')
    tabLinks.linkTabs('src-tab', 'missing-state')
    tabLinks.linkTabs('src-tab', 'agent-tab')
    tabLinks.linkTabs('src-tab', 'already-at-path')

    expect(sync.syncPathFromTab('src-tab', 'same.md')).toBe(true)
  })

  it('does not navigate plex or note tabs when non-note path (e.g. pdf) is selected', () => {
    const tabLinks = useTabLinks()
    tabLinks.linkTabs('files-1', 'plex-1')
    tabLinks.linkTabs('files-1', 'note-1')

    const loadInTabMock = vi.fn()
    const plexGoMock = vi.fn()

    const heldMock: any = {
      getTab: (id: string) => {
        if (id === 'note-1') return { kind: { kind: 'note' } }
        if (id === 'plex-1') return { kind: { kind: 'plex' } }
        return null
      },
      getTabStateIn: (id: string, kind: string) => {
        if (id === 'plex-1' && kind === 'plex') {
          return { view: { here: { value: 'old.md' }, go: plexGoMock } }
        }
        return null
      },
    }

    const editingMock: any = {
      noted: { loadInTab: loadInTabMock },
    }

    const sync = useTabSync({ held: heldMock, tabLinks, editing: editingMock })

    // Non-note selection (e.g. PDF)
    sync.syncPathFromTab('files-1', 'Books/Guide.pdf')

    expect(loadInTabMock).not.toHaveBeenCalled()
    expect(plexGoMock).not.toHaveBeenCalled()
  })
})
