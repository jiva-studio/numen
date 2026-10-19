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
  const held = useWindowTabs()

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

  const tabSync = useTabSync({ held, tabLinks, onError: vi.fn() })

  const plexKind = createPlexKind({
    core: {
      neighbourhood: (path: string) =>
        Promise.resolve({
          focus: { path, title: 'Focus' },
          related: [{ path: neighbourhoodPath, title: neighbourhoodTitle, kind: 'link' }],
        }),
    } as never,
    tabOpeners: tabOpenersMock,
    held,
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

  return { tabLinks, held, navigateNote, revealPathMock, plexKind }
}

describe('createPlexKind with linked tabs', () => {
  it('loads note in linked tab when plex node is opened', async () => {
    const { tabLinks, held, navigateNote, plexKind } = createPlexLinkedFixture(
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

    held.registerKinds([plexKind.kind, noteKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const noteTabId = await held.openTabOfKind('note', 'Note.md')

    tabLinks.linkTabs(plexTabId, noteTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<MockPlexState>(plexTabId, 'plex')
    if (!plexState) throw new Error('plexState expected')
    await plexState.view.go('Start.md')

    const targetNode = plexState.picture.value.nodes.find(
      (n: MockPlexNode) => n.title === 'Target Title',
    )
    expect(targetNode).toBeDefined()
    if (!targetNode) throw new Error('targetNode expected')

    plexState.openNode(targetNode.id)
    expect(navigateNote).toHaveBeenCalledWith(
      held.getTab(noteTabId)?.state,
      'Target.md',
      'Target Title',
    )
  })

  it('reveals file in linked files tab when plex node is activated', async () => {
    const { tabLinks, held, revealPathMock, plexKind } = createPlexLinkedFixture(
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

    held.registerKinds([plexKind.kind, filesKind])

    const plexTabId = await held.openTabOfKind('plex', 'Start.md')
    const filesTabId = await held.openTabOfKind('files', '')

    tabLinks.linkTabs(plexTabId, filesTabId)
    held.handle.show(plexTabId)

    const plexState = held.getTabStateIn<MockPlexState>(plexTabId, 'plex')
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
