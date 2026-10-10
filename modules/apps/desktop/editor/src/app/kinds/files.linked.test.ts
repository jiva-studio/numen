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
    const editingMock: any = {
      notes: { applyPathChanges: vi.fn() },
      making: { createUntitled: vi.fn() },
      noted: { loadInTab: loadInTabMock },
    }

    const tabSync = useTabSync({ held, tabLinks, editing: editingMock })

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
      } as any,
      tabOpeners: {} as any,
      held,
      runs: { canRun: () => true } as any,
      editing: editingMock,
      vaults: { makes: shallowRef(new Map()) } as any,
      getTarget: () => ({}) as any,
      runCommand: () => {},
      commandDeps: () => ({}) as any,
      dragged: shallowRef([]),
      writeMessage: () => {},
      destinations: {} as any,
      tabLinks,
      tabSync,
    })

    const noteKind: any = {
      kind: 'note',
      open: () => ({}),
      getTitle: () => 'My Note',
      pane: {},
    }

    held.registerKinds([filesKind.files.kind, noteKind])

    const filesTabId = await held.openTabOfKind('files', '')
    const noteTabId = await held.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(filesTabId, noteTabId)
    held.handle.show(filesTabId)

    const filesState = held.getTabStateIn<any>(filesTabId, 'files')
    vi.spyOn(filesState.list, 'getEntryAt').mockImplementation(
      (p: string) => entriesMap.get(p) as any,
    )

    filesState.activate('folder/target.md')

    expect(loadInTabMock).toHaveBeenCalledWith(
      noteTabId.replace(/^note:/, ''),
      'folder/target.md',
      'target.md',
    )
  })
})
