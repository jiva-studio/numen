/** The window and the notes of the vault, as a plugin's tabs are handed them. */
import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceHost,
  type WorkspaceHostDeps,
} from './pluginServices'
import type { CommandTarget } from '@/features/command-palette'
import type { NoteCreator } from '@/entities/note'

const createWorkspace = (overrides: Partial<WorkspaceHostDeps> = {}) => {
  const failure = ref('')
  const deps = {
    window: {
      failure,
      opening: ref('Root.md'),
      readInitialNote: vi.fn(() => Promise.resolve('Root.md')),
    },
    dragged: shallowRef<readonly string[]>([]),
    tabOpeners: {
      openFile: vi.fn(() => Promise.resolve()),
      openFileAt: vi.fn(() => Promise.resolve()),
      findFileIcon: vi.fn(() => undefined),
    },
    tabSync: { syncPathFromTab: vi.fn(() => true) },
    getTarget: () => ({ tab: 'plex:1', kind: 'plex', path: 'Here.md' }) as unknown as CommandTarget,
    runCommand: vi.fn(),
    ask: vi.fn(),
    getNoteInFocus: () => ({ path: 'Focus.md', title: 'Focus' }),
    getNoteCommands: () => [{ id: 'plugin:share', text: 'Share the note', group: 'note' }],
    writeMessage: vi.fn(),
    getMessageWriter: () => vi.fn(),
    today: () => '2026-09-04',
    ...overrides,
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

  it('opens a file where and at the line requested, and only what is requested', () => {
    const { workspace, deps } = createWorkspace()

    workspace.openFile('One.md', { title: 'One', placement: 'split', line: 3 })
    workspace.openFile('Two.md')

    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(1, 'One.md', 'One', {
      where: 'split',
      line: 3,
    })
    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(2, 'Two.md', '', {})
  })

  it('opens a file at the selector requested', () => {
    const { workspace, deps } = createWorkspace()

    workspace.openFile('Talk.bin', { selector: { t: [12, 14] } })

    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('Talk.bin', '', {
      selector: { t: [12, 14] },
    })
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

  it('lists the commands the plugins declare over a note', () => {
    const { workspace } = createWorkspace()

    expect(workspace.getNoteCommands()).toStrictEqual([
      { id: 'plugin:share', text: 'Share the note', group: 'note' },
    ])
  })

  it('shares what is dragged, asks the agent and says what failed', () => {
    const { workspace, deps } = createWorkspace()

    workspace.setDraggedPaths(['One.md'])
    workspace.ask('what is One?')
    workspace.showError('the link was not written')
    expect(workspace.getToday()).toBe('2026-09-04')

    expect(deps.dragged.value).toStrictEqual(['One.md'])
    expect(workspace.draggedPaths.value).toStrictEqual(['One.md'])
    expect(deps.ask).toHaveBeenCalledWith('what is One?')
    expect(deps.writeMessage).toHaveBeenCalledWith('the link was not written', 'error')
  })

  it('reads the note the vault opens with', async () => {
    const { workspace } = createWorkspace()

    expect(workspace.openingPath.value).toBe('Root.md')
    expect(await workspace.readOpeningPath()).toBe('Root.md')
  })
})

describe('the window a plugin opens files and reads the note in focus through', () => {
  it('opens a file with the runs of its text marked', () => {
    const { workspace, deps } = createWorkspace()

    workspace.openFileAt('Source.pdf', [{ start: 3, end: 9 }])

    expect(deps.tabOpeners.openFileAt).toHaveBeenCalledWith('Source.pdf', [{ start: 3, end: 9 }])
  })

  it('says why a file could not be opened with its runs marked', async () => {
    const { workspace, deps } = createWorkspace({
      tabOpeners: {
        openFile: vi.fn(() => Promise.resolve()),
        openFileAt: vi.fn(() => Promise.reject(new Error('no such file'))),
        findFileIcon: vi.fn(() => undefined),
      },
    })

    workspace.openFileAt('Gone.pdf', [])
    await new Promise((isDone) => setTimeout(isDone, 0))

    expect(deps.writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('names the note the window relies on', () => {
    const { workspace } = createWorkspace()

    expect(workspace.getNoteInFocus()).toStrictEqual({ path: 'Focus.md', title: 'Focus' })
  })
})

describe('the agent a plugin is handed', () => {
  const create = () => {
    const port = { ask: vi.fn(), finish: vi.fn(() => Promise.resolve()) }
    const core = {
      resolveLinks: vi.fn(() =>
        Promise.resolve([{ raw: '[[A]]', address: 'name://A', text: 'A', path: 'A.md' }]),
      ),
    }
    const rest = {
      getSetting: vi.fn(() => 'zeta'),
      writeSetting: vi.fn(() => Promise.resolve()),
      getModelsAt: vi.fn(() => []),
    }
    const unreachable = ref('')
    const agent = createAgentHost({ port, core, window: { unreachable }, settings: { rest } })
    return { agent, port, core, rest, unreachable }
  }

  it('asks and ends a conversation through the port', async () => {
    const { agent, port } = create()
    const query = {
      question: 'q',
      focus: '',
      conversation: 'c:1',
      signal: new AbortController().signal,
    }

    agent.ask(query)
    await agent.finish('c:1')

    expect(port.ask).toHaveBeenCalledWith(query)
    expect(port.finish).toHaveBeenCalledWith('c:1')
  })

  it('resolves the wikilinks of an answer from the top of the vault', async () => {
    const { agent, core } = create()

    expect(await agent.resolveLinks('See [[A]].')).toStrictEqual([
      { raw: '[[A]]', address: 'name://A', text: 'A', path: 'A.md' },
    ])
    expect(core.resolveLinks).toHaveBeenCalledWith('', 'See [[A]].', undefined)
  })

  it('resolves the wikilinks of a text in the vault a plugin names', async () => {
    const { agent, core } = create()

    await agent.resolveLinks('See [[A]].', 'garden')

    expect(core.resolveLinks).toHaveBeenCalledWith('', 'See [[A]].', 'garden')
  })

  it('says why the agent cannot be reached while the window says it', () => {
    const { agent, unreachable } = create()

    expect(agent.getUnreachable()).toBe('')
    unreachable.value = 'down'
    expect(agent.getUnreachable()).toBe('down')
  })

  it('reads and writes settings and lists models through the settings of the window', async () => {
    const { agent, rest } = create()

    await agent.writeSetting(['agent', 'use'], 'omega')

    expect(agent.getSetting(['agent', 'use'])).toBe('zeta')
    expect(agent.getModelsAt(['agent', 'zeta', 'model'])).toStrictEqual([])
    expect(rest.writeSetting).toHaveBeenCalledWith(['agent', 'use'], 'omega')
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
    const pluginNotes = {
      readNote: vi.fn(() => Promise.resolve({ body: 'Prose.' })),
      writeNote: vi.fn(() => Promise.resolve()),
    }
    return {
      making,
      core,
      search,
      pluginNotes,
      notes: createVaultNotes({ core, search, making, pluginNotes }),
    }
  }

  it('reads and writes the prose of a note through the queue of the plugins', async () => {
    const { notes, pluginNotes } = createNotes()

    expect(await notes.read('Root.md')).toStrictEqual({ body: 'Prose.' })
    await notes.write('Root.md', 'More prose.')

    expect(pluginNotes.readNote).toHaveBeenCalledWith('Root.md')
    expect(pluginNotes.writeNote).toHaveBeenCalledWith('Root.md', 'More prose.')
  })

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
