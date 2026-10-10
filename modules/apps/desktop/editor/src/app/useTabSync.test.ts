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
        if (id === 'note-1') return { kind: { kind: 'note' } }
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
    expect(loadInTabMock).toHaveBeenCalledWith('note-1', 'test.md', 'Test')
    expect(revealPathMock).toHaveBeenCalledWith('test.md')

    // Sync from Files -> Plex & Note
    const syncedFromFiles = sync.syncPathFromTab('files-1', 'other.md')
    expect(syncedFromFiles).toBe(true)
    expect(loadInTabMock).toHaveBeenCalledWith('note-1', 'other.md', '')
    expect(plexGoMock).toHaveBeenCalledWith('other.md')
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
