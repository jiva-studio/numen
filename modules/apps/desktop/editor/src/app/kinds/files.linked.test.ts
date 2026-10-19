import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { createFilesKind } from './files'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from '../useTabSync'
import { useWindowTabs } from '@/entities/tab'

describe('createFilesKind with linked tabs', () => {
  it('syncs note when destination is activated in linked tab', async () => {
    const tabLinks = useTabLinks()
    const held = useWindowTabs()

    const loadInTabMock = vi.fn()
    const editingMock = {
      notes: { applyPathChanges: vi.fn() },
      making: { createUntitled: vi.fn() },
    } as never

    const tabSync = useTabSync({ held, tabLinks })

    const entriesMap = new Map([
      [
        'folder/target.md',
        { path: 'folder/target.md', name: 'target.md', isFolder: false, kind: 'note' as const },
      ],
    ])

    const filesKind = createFilesKind({
      core: {
        listDirectory: async () => ({
          ok: true,
          value: {
            entries: [
              { path: 'folder/target.md', name: 'target.md', isFolder: false, kind: 'note' },
            ],
          },
        }),
        createUrl: vi.fn(),
      } as never,
      tabOpeners: {} as never,
      held,
      runs: { canRun: () => true } as never,
      editing: editingMock,
      vaults: { makes: shallowRef(new Map()) } as never,
      getTarget: () => ({}) as never,
      runCommand: () => {},
      commandDeps: () => ({}) as never,
      dragged: shallowRef([]),
      writeMessage: () => {},
      destinations: {} as never,
      tabLinks,
      tabSync,
    })

    const noteKind = {
      kind: 'note',
      open: () => ({}),
      getTitle: () => 'My Note',
      pane: {},
      navigate: loadInTabMock,
    } as never

    held.registerKinds([filesKind.files.kind, noteKind])

    const filesTabId = await held.openTabOfKind('files', '')
    const noteTabId = await held.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(filesTabId, noteTabId)
    held.handle.show(filesTabId)

    const filesState = held.getTabStateIn<{
      list: { getEntryAt: (p: string) => unknown }
      activate: (p: string) => void
    }>(filesTabId, 'files')
    if (!filesState) throw new Error('filesState expected')
    vi.spyOn(filesState.list, 'getEntryAt').mockImplementation(
      ((p: string) => entriesMap.get(p) as unknown) as never,
    )

    filesState.activate('folder/target.md')

    expect(loadInTabMock).toHaveBeenCalledWith(
      held.getTab(noteTabId)?.state,
      'folder/target.md',
      'target.md',
    )
  })
})
