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

const createPlexLinkedFixture = (neighbourhoodPath: string, neighbourhoodTitle: string) => {
  const tabLinks = useTabLinks()
  const windowTabs = useWindowTabs()

  const navigateNote = vi.fn()
  const openFileMock = vi.fn()
  const revealPathMock = vi.fn()

  const editingMock = {
    making: {},
  } as never

  const tabOpenersMock = {
    openFile: openFileMock,
  } as never

  const windowMock = {
    failure: ref(false),
    opening: ref(''),
    readInitialNote: vi.fn(() => Promise.resolve('')),
  } as never

  const settingsMock = {
    nesting: {
      isNesting: ref(false),
      headings: ref(0),
    },
  } as never

  const tabSync = useTabSync({ windowTabs, tabLinks, onError: vi.fn() })

  const plexKind = createPlexKind({
    core: {
      neighbourhood: (path: string) =>
        Promise.resolve({
          focus: { path, title: 'Focus' },
          related: [{ path: neighbourhoodPath, title: neighbourhoodTitle, kind: 'link' }],
        }),
    } as never,
    tabOpeners: tabOpenersMock,
    held: windowTabs,
    editing: editingMock,
    settings: settingsMock,
    window: windowMock,
    getTarget: () => ({}) as never,
    runCommand: vi.fn(),
    dragged: shallowRef([]),
    writeMessage: vi.fn(),
    askAgent: vi.fn(),
    tabLinks,
    tabSync,
  })

  return { tabLinks, windowTabs, navigateNote, revealPathMock, plexKind }
}

describe('createPlexKind with linked tabs', () => {
  it('loads note in linked tab when plex node is opened', async () => {
    const { tabLinks, windowTabs, navigateNote, plexKind } = createPlexLinkedFixture(
      'Target.md',
      'Target Title',
    )

    const noteKind = {
      kind: 'note',
      open: () => ({}),
      getTitle: () => 'My Note',
      pane: {},
      navigate: navigateNote,
    } as never

    windowTabs.registerKinds([plexKind.kind, noteKind])

    const plexTabId = await windowTabs.openTabOfKind('plex', 'Start.md')
    const noteTabId = await windowTabs.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(plexTabId, noteTabId)
    windowTabs.handle.show(plexTabId)

    const plexState = windowTabs.getTabStateIn<MockPlexState>(plexTabId, 'plex')
    if (!plexState) throw new Error('plexState expected')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find(
      (n: MockPlexNode) => n.title === 'Target Title',
    )
    expect(targetNode).toBeDefined()
    if (!targetNode) throw new Error('targetNode expected')

    plexState.openNode(targetNode.id)
    expect(navigateNote).toHaveBeenCalledWith(
      windowTabs.getTab(noteTabId)?.state,
      'Target.md',
      'Target Title',
    )
  })

  it('reveals file in linked files tab when plex node is activated', async () => {
    const { tabLinks, windowTabs, revealPathMock, plexKind } = createPlexLinkedFixture(
      'Selected.md',
      'Selected Title',
    )

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
      navigate: (state: typeof filesStateMock, path: string) => state.list.revealPath(path),
    } as never

    windowTabs.registerKinds([plexKind.kind, filesKind])

    const plexTabId = await windowTabs.openTabOfKind('plex', 'Start.md')
    const filesTabId = await windowTabs.openTabOfKind('files', '')

    tabLinks.linkTabs(plexTabId, filesTabId)
    windowTabs.handle.show(plexTabId)

    const plexState = windowTabs.getTabStateIn<MockPlexState>(plexTabId, 'plex')
    if (!plexState) throw new Error('plexState expected')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find(
      (n: MockPlexNode) => n.title === 'Selected Title',
    )
    expect(targetNode).toBeDefined()
    if (!targetNode) throw new Error('targetNode expected')

    plexState.activate(targetNode.id)
    expect(revealPathMock).toHaveBeenCalledWith('Selected.md')
  })
})
