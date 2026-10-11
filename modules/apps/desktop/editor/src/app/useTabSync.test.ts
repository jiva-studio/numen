import { describe, expect, it, vi } from 'vitest'
import { useTabLinks } from '@/features/tab-linking'
import { isNotePath, useTabSync } from './useTabSync'

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

const linkedTo = (...targets: string[]): any => ({ getLinkedTargets: () => targets })

describe('isNotePath', () => {
  it('holds for markdown files and for paths without an extension', () => {
    expect(isNotePath('Ideas/Plan.md')).toBe(true)
    expect(isNotePath('Ideas/Plan')).toBe(true)
  })

  it('does not hold for other extensions', () => {
    expect(isNotePath('Books/Guide.pdf')).toBe(false)
    expect(isNotePath('Ideas/Plan.txt')).toBe(false)
  })
})

describe('useTabSync targets', () => {
  it('loads the note under the id its tab state holds', () => {
    const loadInTab = vi.fn()
    const held: any = { getTab: () => ({ kind: { kind: 'note' }, state: { id: 'kept' } }) }

    useTabSync({
      held,
      tabLinks: linkedTo('note:other'),
      editing: { noted: { loadInTab } } as any,
    }).syncPathFromTab('s', 'a.md', 'A')

    expect(loadInTab).toHaveBeenCalledWith('kept', 'a.md', 'A')
  })

  it('loads a stateless note tab under its id without the leading note:', () => {
    const loadInTab = vi.fn()
    const held: any = { getTab: () => ({ kind: { kind: 'note' } }) }

    useTabSync({
      held,
      tabLinks: linkedTo('note:abc', 'x-note:def'),
      editing: { noted: { loadInTab } } as any,
    }).syncPathFromTab('s', 'a.md')

    expect(loadInTab).toHaveBeenCalledWith('abc', 'a.md', '')
    expect(loadInTab).toHaveBeenCalledWith('x-note:def', 'a.md', '')
  })

  it('leaves a plex tab alone when it already shows the path', () => {
    const go = vi.fn()
    const held: any = {
      getTab: () => ({ kind: { kind: 'plex' } }),
      getTabStateIn: () => ({ view: { here: { value: 'same.md' }, go } }),
    }
    const sync = useTabSync({ held, tabLinks: linkedTo('p'), editing: {} as any })

    sync.syncPathFromTab('s', 'same.md')
    expect(go).not.toHaveBeenCalled()

    sync.syncPathFromTab('s', 'next.md')
    expect(go).toHaveBeenCalledWith('next.md')
  })

  it('skips a files tab that has no state', () => {
    const held: any = { getTab: () => ({ kind: { kind: 'files' } }), getTabStateIn: () => null }
    const sync = useTabSync({ held, tabLinks: linkedTo('f'), editing: {} as any })

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
  })
})

describe('useTabSync guards', () => {
  it('refuses an empty source tab id or an empty path even when targets are linked', () => {
    const loadInTab = vi.fn()
    const held: any = { getTab: () => ({ kind: { kind: 'note' } }) }
    const sync = useTabSync({
      held,
      tabLinks: linkedTo('n'),
      editing: { noted: { loadInTab } } as any,
    })

    expect(sync.syncPathFromTab('', 'a.md')).toBe(false)
    expect(sync.syncPathFromTab('s', '')).toBe(false)
    expect(loadInTab).not.toHaveBeenCalled()
  })

  it('refuses a sync started while one is running, and accepts the next after it ends', () => {
    const inner: boolean[] = []
    const held: any = { getTab: () => ({ kind: { kind: 'note' } }) }
    const loadInTab = vi.fn(() => {
      inner.push(sync.syncPathFromTab('s', 'b.md'))
    })
    const sync = useTabSync({
      held,
      tabLinks: linkedTo('n'),
      editing: { noted: { loadInTab } } as any,
    })

    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
    expect(inner).toEqual([false])
    expect(loadInTab).toHaveBeenCalledTimes(1)

    expect(sync.syncPathFromTab('s', 'c.md')).toBe(true)
    expect(loadInTab).toHaveBeenCalledTimes(2)
  })

  it('ends the running state when a target throws', () => {
    const held: any = { getTab: () => ({ kind: { kind: 'note' } }) }
    let fail = true
    const loadInTab = vi.fn(() => {
      if (fail) throw new Error('boom')
    })
    const sync = useTabSync({
      held,
      tabLinks: linkedTo('n'),
      editing: { noted: { loadInTab } } as any,
    })

    expect(() => sync.syncPathFromTab('s', 'a.md')).toThrow('boom')
    fail = false
    expect(sync.syncPathFromTab('s', 'a.md')).toBe(true)
  })
})
