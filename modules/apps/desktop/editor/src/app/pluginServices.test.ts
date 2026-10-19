/** The window and the notes of the vault, as a plugin's tabs are handed them. */
import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { createVaultNotes, createWorkspaceHost, type WorkspaceHostDeps } from './pluginServices'
import type { CommandTarget } from '@/features/command-palette'
import type { NoteCreator } from '@/entities/note'

const createWorkspace = (over: Partial<WorkspaceHostDeps> = {}) => {
  const failure = ref('')
  const deps = {
    window: {
      failure,
      opening: ref('Root.md'),
      readInitialNote: vi.fn(() => Promise.resolve('Root.md')),
    },
    nesting: { isNesting: ref(true), headings: ref(3) },
    dragged: shallowRef<readonly string[]>([]),
    tabOpeners: { openFile: vi.fn(() => Promise.resolve()) },
    tabSync: { syncPathFromTab: vi.fn(() => true) },
    getTarget: () => ({ tab: 'plex:1', kind: 'plex', path: 'Here.md' }) as unknown as CommandTarget,
    runCommand: vi.fn(),
    ask: vi.fn(),
    writeMessage: vi.fn(),
    runs: { canRun: (run: string) => run !== 'transcribe' },
    vaults: { makes: { value: new Map([['Lecture.mp3', { transcript: 'done' as const }]]) } },
    ...over,
  } satisfies WorkspaceHostDeps
  return { deps, failure, workspace: createWorkspaceHost(deps) }
}

describe('the window a plugin is handed', () => {
  it('is ready while it says nothing failed', () => {
    const { workspace, failure } = createWorkspace()

    expect(workspace.isReady.value).toBe(true)
    failure.value = 'the vault is gone'
    expect(workspace.isReady.value).toBe(false)
  })

  it('opens a file where and at the line asked, and only what is asked', () => {
    const { workspace, deps } = createWorkspace()

    workspace.openFile('One.md', { title: 'One', placement: 'beside', line: 3 })
    workspace.openFile('Two.md')

    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(1, 'One.md', 'One', {
      where: 'beside',
      line: 3,
    })
    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(2, 'Two.md', '', {})
  })

  it('moves the linked tabs, and says no tab moved where the window links none', () => {
    const { workspace, deps } = createWorkspace()

    expect(workspace.syncPath('plex:1', 'One.md', 'One')).toBe(true)
    expect(deps.tabSync.syncPathFromTab).toHaveBeenCalledWith('plex:1', 'One.md', 'One')

    const { tabSync: _tabSync, ...unlinked } = deps
    expect(createWorkspaceHost(unlinked).syncPath('plex:1', 'One.md')).toBe(false)
  })

  it('runs a command over the note named, from the tab in front', () => {
    const { workspace, deps } = createWorkspace()

    workspace.runCommand('rename', { path: 'One.md', title: 'One' })

    expect(deps.runCommand).toHaveBeenCalledWith('rename', {
      tab: 'plex:1',
      kind: 'plex',
      path: 'One.md',
      title: 'One',
    })
  })

  it('shares what is dragged, asks the agent and says what failed', () => {
    const { workspace, deps } = createWorkspace()

    workspace.setDraggedPaths(['One.md'])
    workspace.ask('what is One?')
    workspace.showError('the link was not written')

    expect(deps.dragged.value).toStrictEqual(['One.md'])
    expect(workspace.draggedPaths.value).toStrictEqual(['One.md'])
    expect(deps.ask).toHaveBeenCalledWith('what is One?')
    expect(deps.writeMessage).toHaveBeenCalledWith('the link was not written', 'error')
  })

  it('reads the note the vault opens with, and how headings are shown', async () => {
    const { workspace } = createWorkspace()

    expect(workspace.openingPath.value).toBe('Root.md')
    expect(await workspace.readOpeningPath()).toBe('Root.md')
    expect(workspace.headingDisplay.isNesting.value).toBe(true)
    expect(workspace.headingDisplay.count.value).toBe(3)
  })
})

describe('the notes of the vault a plugin is handed', () => {
  const createNotes = () => {
    const making = {
      createUntitled: vi.fn(() => Promise.resolve({ path: 'Untitled.md', title: 'Untitled' })),
      createInRelation: vi.fn(() => Promise.resolve({ path: 'Made.md', title: 'Made' })),
      createWithTitle: vi.fn(() => Promise.resolve(null)),
      join: vi.fn(() => Promise.resolve(true)),
      replaceLinks: vi.fn(() => Promise.resolve(false)),
    } as unknown as NoteCreator
    const core = {
      neighbourhood: vi.fn((path: string) =>
        Promise.resolve({
          focus: { path, title: 'Root' },
          focusType: 'note' as const,
          related: [],
        }),
      ),
      headings: vi.fn(() => Promise.resolve(new Map())),
    }
    const search = {
      names: vi.fn(() =>
        Promise.resolve([
          { path: 'One.md', title: 'One', heading: '', line: -1, at: [], type: 'note' as const },
        ]),
      ),
    }
    return { making, core, search, notes: createVaultNotes({ core, search, making }) }
  }

  it('reads what is around a note and the headings of notes', async () => {
    const { notes, core } = createNotes()

    expect((await notes.getNeighbourhood('Root.md')).focus).toStrictEqual({
      path: 'Root.md',
      title: 'Root',
    })
    await notes.getHeadings(['Root.md'])
    expect(core.headings).toHaveBeenCalledWith(['Root.md'])
  })

  it('finds notes by name, twenty unless told how many', async () => {
    const { notes, search } = createNotes()

    expect(await notes.findNotes('On')).toStrictEqual([{ path: 'One.md', title: 'One' }])
    await notes.findNotes('On', 5)

    expect(search.names).toHaveBeenNthCalledWith(1, 'On', 20)
    expect(search.names).toHaveBeenNthCalledWith(2, 'On', 5)
  })

  it('makes notes through the note maker, untitled at the root of the vault', async () => {
    const { notes, making } = createNotes()

    expect(await notes.createUntitledNote()).toBe('Untitled.md')
    expect(making.createUntitled).toHaveBeenCalledWith('', [])
    expect(await notes.createInRelation('Root.md', 'child')).toStrictEqual({
      path: 'Made.md',
      title: 'Made',
    })
    expect(await notes.createWithTitle('Two', 'Root.md', 'parent')).toBeNull()
    expect(await notes.join('Root.md', 'Two.md', 'jump', 'see')).toBe(true)
    expect(making.join).toHaveBeenCalledWith('Root.md', 'Two.md', 'jump', 'see')
    expect(await notes.replaceLinks('Root.md', 'Two.md', [])).toBe(false)
    expect(notes.creatableRelations).toStrictEqual(['parent', 'child', 'jump'])
  })

  it('makes no path where the note maker made no untitled note', async () => {
    const { notes, making } = createNotes()
    vi.mocked(making.createUntitled).mockResolvedValueOnce(null)

    expect(await notes.createUntitledNote()).toBe('')
  })
})
