import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { createPlexKind } from './plex'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from '../useTabSync'
import { useWindowTabs } from '@/entities/tab'

describe('createPlexKind with linked tabs', () => {
  it('loads note in linked tab when plex node is opened', async () => {
    const tabLinks = useTabLinks()
    const held = useWindowTabs()

    const loadInTabMock = vi.fn()
    const openFileMock = vi.fn()

    const editingMock: any = {
      making: {},
      noted: {
        loadInTab: loadInTabMock,
      },
    }

    const tabOpenersMock: any = {
      openFile: openFileMock,
    }

    const windowMock: any = {
      failure: ref(false),
      opening: ref(''),
      readInitialNote: async () => '',
    }

    const settingsMock: any = {
      hungParts: {
        isHanging: ref(false),
        parts: ref(0),
      },
    }

    const tabSync = useTabSync({ held, tabLinks, editing: editingMock })

    const plexKind = createPlexKind({
      core: {
        neighbourhood: async (path: string) => ({
          focus: { path, title: 'Focus Title' },
          related: [{ path: 'Target.md', title: 'Target Title', kind: 'link' }],
        }),
      } as any,
      tabOpeners: tabOpenersMock,
      held,
      editing: editingMock,
      settings: settingsMock,
      window: windowMock,
      getTarget: () => ({}) as any,
      runCommand: () => {},
      dragged: shallowRef([]),
      writeMessage: () => {},
      askAgent: () => {},
      tabLinks,
      tabSync,
    })

    const noteKind: any = {
      kind: 'note',
      open: () => ({}),
      getTitle: () => 'My Note',
      pane: {},
    }

    held.registerKinds([plexKind.kind, noteKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const noteTabId = await held.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(plexTabId, noteTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<any>(plexTabId, 'plex')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find((n: any) => n.title === 'Target Title')
    expect(targetNode).toBeDefined()

    plexState.openNode(targetNode.id)
    expect(loadInTabMock).toHaveBeenCalledWith(
      noteTabId.replace(/^note:/, ''),
      'Target.md',
      'Target Title',
    )
  })

  it('reveals file in linked files tab when plex node is activated', async () => {
    const tabLinks = useTabLinks()
    const held = useWindowTabs()

    const revealPathMock = vi.fn()

    const editingMock: any = {
      making: {},
      noted: {
        loadInTab: vi.fn(),
      },
    }

    const tabOpenersMock: any = {
      openFile: vi.fn(),
    }

    const windowMock: any = {
      failure: ref(false),
      opening: ref(''),
      readInitialNote: async () => '',
    }

    const settingsMock: any = {
      hungParts: {
        isHanging: ref(false),
        parts: ref(0),
      },
    }

    const tabSync = useTabSync({ held, tabLinks, editing: editingMock })

    const plexKind = createPlexKind({
      core: {
        neighbourhood: async (path: string) => ({
          focus: { path, title: 'Focus' },
          related: [{ path: 'Selected.md', title: 'Selected Title', kind: 'link' }],
        }),
      } as any,
      tabOpeners: tabOpenersMock,
      held,
      editing: editingMock,
      settings: settingsMock,
      window: windowMock,
      getTarget: () => ({}) as any,
      runCommand: () => {},
      dragged: shallowRef([]),
      writeMessage: () => {},
      askAgent: () => {},
      tabLinks,
      tabSync,
    })

    const filesStateMock = {
      list: {
        revealPath: revealPathMock,
      },
    }

    const filesKind: any = {
      kind: 'files',
      open: () => filesStateMock,
      getTitle: () => 'Files',
      pane: {},
    }

    held.registerKinds([plexKind.kind, filesKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const filesTabId = await held.openTabOfKind('files', '')

    tabLinks.linkTabs(plexTabId, filesTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<any>(plexTabId, 'plex')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find((n: any) => n.title === 'Selected Title')
    expect(targetNode).toBeDefined()

    plexState.activate(targetNode.id)
    expect(revealPathMock).toHaveBeenCalledWith('Selected.md')
  })
})
