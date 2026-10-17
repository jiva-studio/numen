import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { createPlexKind } from './plex'
import { useTabLinks } from '@/features/tab-linking'
import { useTabSync } from '../useTabSync'
import { useWindowTabs } from '@/entities/tab'

interface MockPlexNode {
  id: string
  title: string
}

interface MockPlexState {
  view: { go: (p: string) => Promise<void> }
  picture: { value: { nodes: MockPlexNode[] } }
  openNode: (id: string) => void
  activate: (id: string) => void
}

describe('createPlexKind with linked tabs', () => {
  it('loads note in linked tab when plex node is opened', async () => {
    const tabLinks = useTabLinks()
    const held = useWindowTabs()

    const loadInTabMock = vi.fn()
    const openFileMock = vi.fn()

    const editingMock = {
      making: {},
      noted: {
        loadInTab: loadInTabMock,
      },
    } as never

    const tabOpenersMock = {
      openFile: openFileMock,
    } as never

    const windowMock = {
      failure: ref(false),
      opening: ref(''),
      readInitialNote: async () => '',
    } as never

    const settingsMock = {
      nesting: {
        isNesting: ref(false),
        headings: ref(0),
      },
    } as never

    const tabSync = useTabSync({ held, tabLinks, editing: editingMock })

    const plexKind = createPlexKind({
      core: {
        neighbourhood: async (path: string) => ({
          focus: { path, title: 'Focus Title' },
          related: [{ path: 'Target.md', title: 'Target Title', kind: 'link' }],
        }),
      } as never,
      tabOpeners: tabOpenersMock,
      held,
      editing: editingMock,
      settings: settingsMock,
      window: windowMock,
      getTarget: () => ({}) as never,
      runCommand: () => {},
      dragged: shallowRef([]),
      writeMessage: () => {},
      askAgent: () => {},
      tabLinks,
      tabSync,
    })

    const noteKind = {
      kind: 'note',
      open: () => ({}),
      getTitle: () => 'My Note',
      pane: {},
    } as never

    held.registerKinds([plexKind.kind, noteKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const noteTabId = await held.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(plexTabId, noteTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<MockPlexState>(plexTabId, 'plex')
    if (!plexState) throw new Error('plexState expected')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find((n: MockPlexNode) => n.title === 'Target Title')
    expect(targetNode).toBeDefined()
    if (!targetNode) throw new Error('targetNode expected')

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

    const editingMock = {
      making: {},
      noted: {
        loadInTab: vi.fn(),
      },
    } as never

    const tabOpenersMock = {
      openFile: vi.fn(),
    } as never

    const windowMock = {
      failure: ref(false),
      opening: ref(''),
      readInitialNote: async () => '',
    } as never

    const settingsMock = {
      nesting: {
        isNesting: ref(false),
        headings: ref(0),
      },
    } as never

    const tabSync = useTabSync({ held, tabLinks, editing: editingMock })

    const plexKind = createPlexKind({
      core: {
        neighbourhood: async (path: string) => ({
          focus: { path, title: 'Focus' },
          related: [{ path: 'Selected.md', title: 'Selected Title', kind: 'link' }],
        }),
      } as never,
      tabOpeners: tabOpenersMock,
      held,
      editing: editingMock,
      settings: settingsMock,
      window: windowMock,
      getTarget: () => ({}) as never,
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

    const filesKind = {
      kind: 'files',
      open: () => filesStateMock,
      getTitle: () => 'Files',
      pane: {},
    } as never

    held.registerKinds([plexKind.kind, filesKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const filesTabId = await held.openTabOfKind('files', '')

    tabLinks.linkTabs(plexTabId, filesTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<MockPlexState>(plexTabId, 'plex')
    if (!plexState) throw new Error('plexState expected')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find((n: MockPlexNode) => n.title === 'Selected Title')
    expect(targetNode).toBeDefined()
    if (!targetNode) throw new Error('targetNode expected')

    plexState.activate(targetNode.id)
    expect(revealPathMock).toHaveBeenCalledWith('Selected.md')
  })
})
