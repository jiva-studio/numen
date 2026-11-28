/** The window and the notes of the vault, as a plugin's tabs are handed them. */
import { describe, expect, it, vi } from 'vitest'
import { ReportedError } from '@numen/result'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { ref, shallowRef } from 'vue'
import type { FileOpeners } from '@/entities/tab'
import { createIconHost } from './fileServices'
import {
  createVaultAccess,
  createWindowService,
  createWorkspaceService,
} from './createPluginServices'
import {
  createVaultNotes,
  createWorkspaceFiles,
  createWorkspaceHost,
  createWorkspaceMessages,
  type WorkspaceFilesDeps,
  type WorkspaceHostDeps,
} from './pluginServices'
import { createTestTabs } from '@/testing/tabs'
import { blank } from '@/testing/stubs'
import type { NoteCreator } from '@/entities/note'
import type { MessageWriter } from '@/shared/model/notices/messages'

interface WorkspaceParts extends Pick<WorkspaceFilesDeps, 'writeMessage'> {
  readonly tabOpeners: WorkspaceFilesDeps['tabOpeners'] & Pick<FileOpeners, 'findFileIcon'>
  readonly window: WorkspaceHostDeps['window']
  readonly dragged: WorkspaceHostDeps['dragged']
  readonly tabSync?: WorkspaceFilesDeps['tabSync']
  readonly getNoteInFocus: WorkspaceHostDeps['getNoteInFocus']
  readonly getMessageWriter: (name: string) => WorkspaceFilesDeps['writeMessage']
  readonly today: WorkspaceHostDeps['getToday']
}

const createWorkspace = (overrides: Partial<WorkspaceParts> = {}) => {
  const failure = ref('')
  const deps = {
    window: {
      health: { failure },
      opening: { path: ref('Root.md'), read: vi.fn(() => Promise.resolve('Root.md')) },
    },
    dragged: shallowRef<readonly string[]>([]),
    tabOpeners: {
      openFile: vi.fn(() => Promise.resolve()),
      openFileAt: vi.fn(() => Promise.resolve()),
      findFileIcon: vi.fn(() => undefined),
    },
    tabSync: { syncPathFromTab: vi.fn(() => true) },
    getNoteInFocus: () => ({ path: 'Focus.md', title: 'Focus' }),
    writeMessage: vi.fn(),
    getMessageWriter: () => vi.fn(),
    today: () => '2026-09-04',
    ...overrides,
  } satisfies WorkspaceParts
  return { deps, failure, workspace: createWorkspaceFromParts(deps) }
}

const createWorkspaceFromParts = (parts: WorkspaceParts) =>
  createWorkspaceHost({
    window: parts.window,
    dragged: parts.dragged,
    files: createWorkspaceFiles({
      tabOpeners: parts.tabOpeners,
      ...(parts.tabSync ? { tabSync: parts.tabSync } : {}),
      writeMessage: parts.writeMessage,
    }),
    icons: createIconHost({ findFileIcon: parts.tabOpeners.findFileIcon }),
    messages: createWorkspaceMessages({
      writeMessage: parts.writeMessage,
      getMessageWriter: parts.getMessageWriter,
    }),
    getNoteInFocus: parts.getNoteInFocus,
    getToday: parts.today,
  })

describe('the window a plugin is handed', () => {
  it('is ready while it says nothing failed', () => {
    const { workspace, failure } = createWorkspace()

    expect(workspace.window.isReady.value).toBe(true)
    failure.value = 'the vault is gone'
    expect(workspace.window.isReady.value).toBe(false)
  })

  it('opens a file where and at the line requested, and only what is requested', () => {
    const { workspace, deps } = createWorkspace()

    workspace.files.open('One.md', { title: 'One', placement: 'split', line: 3 })
    workspace.files.open('Two.md')

    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(1, 'One.md', 'One', {
      where: 'split',
      line: 3,
    })
    expect(deps.tabOpeners.openFile).toHaveBeenNthCalledWith(2, 'Two.md', '', {})
  })

  it('opens a file at the selector requested', () => {
    const { workspace, deps } = createWorkspace()

    workspace.files.open('Talk.bin', { selector: { t: [12, 14] } })

    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('Talk.bin', '', {
      selector: { t: [12, 14] },
    })
  })

  it('moves the linked tabs, and says no tab moved where the window links none', () => {
    const { workspace, deps } = createWorkspace()

    expect(workspace.files.syncPath('plex:1', 'One.md', 'One')).toBe(true)
    expect(deps.tabSync?.syncPathFromTab).toHaveBeenCalledWith('plex:1', 'One.md', 'One')

    const { tabSync: _tabSync, ...unlinked } = deps
    expect(createWorkspaceFromParts(unlinked).files.syncPath('plex:1', 'One.md')).toBe(false)
  })

  it('runs no command itself: a plugin runs one through the commands of its context', () => {
    const { workspace } = createWorkspace()

    expect(workspace).not.toHaveProperty('runCommand')
  })

  it('shares what is dragged and says what failed', () => {
    const { workspace, deps } = createWorkspace()

    workspace.dragged.setPaths(['One.md'])
    workspace.messages.showError('the link was not written')
    const customWriter = vi.fn()
    const { workspace: withWriter } = createWorkspace({
      getMessageWriter: (name) => (name === 'custom' ? customWriter : vi.fn()),
    })
    withWriter.messages.getWriter('custom')('hello', 'info')
    expect(customWriter).toHaveBeenCalledWith('hello', 'info')
    expect(workspace.window.getToday()).toBe('2026-09-04')

    expect(deps.dragged.value).toStrictEqual(['One.md'])
    expect(workspace.dragged.paths.value).toStrictEqual(['One.md'])
    expect(deps.writeMessage).toHaveBeenCalledWith('the link was not written', 'error')
  })

  it('reads the note the vault opens with', async () => {
    const { workspace } = createWorkspace()

    expect(workspace.window.openingPath.value).toBe('Root.md')
    expect(await workspace.window.readOpeningPath()).toBe('Root.md')
  })
})

describe('the window a plugin opens files and reads the note in focus through', () => {
  it('opens a file with the runs of its text marked', () => {
    const { workspace, deps } = createWorkspace()

    workspace.files.openAt('Source.pdf', [{ start: 3, end: 9 }])

    expect(deps.tabOpeners.openFileAt).toHaveBeenCalledWith('Source.pdf', [{ start: 3, end: 9 }])
  })

  it('says why a file could not be opened with its runs marked or directly', async () => {
    const { workspace, deps } = createWorkspace({
      tabOpeners: {
        openFile: vi.fn(() => Promise.reject(new Error('open failed'))),
        openFileAt: vi.fn(() => Promise.reject(new Error('no such file'))),
        findFileIcon: vi.fn(() => undefined),
      },
    })

    workspace.files.open('Gone.pdf')
    workspace.files.openAt('Gone.pdf', [])
    await new Promise((isDone) => setTimeout(isDone, 0))

    expect(deps.writeMessage).toHaveBeenCalledTimes(2)
    expect(deps.writeMessage).toHaveBeenNthCalledWith(1, expect.any(String), 'error')
    expect(deps.writeMessage).toHaveBeenNthCalledWith(2, expect.any(String), 'error')
  })

  it('offers a plugin no commands over a row of the vault', () => {
    const { workspace } = createWorkspace()

    expect(workspace).not.toHaveProperty('getPathCommands')
    expect(workspace).not.toHaveProperty('runPathCommand')
  })

  it('names the note the window relies on', () => {
    const { workspace } = createWorkspace()

    expect(workspace.getNoteInFocus()).toStrictEqual({ path: 'Focus.md', title: 'Focus' })
  })
})

describe('the notes of the vault a plugin is handed', () => {
  const createNotes = () => {
    const making = {
      createUntitledNote: vi.fn(() => Promise.resolve('Untitled.md')),
      createWithTitle: vi.fn(() => Promise.resolve(null)),
    } as unknown as NoteCreator
    const listHeadings = vi.fn((_request: object) => ({ headings: [] }))
    const listFileKinds = vi.fn((request: { paths: string[] }) => ({
      kinds: request.paths
        .filter((path) => path.startsWith('links/'))
        .map((path) => ({
          path,
          isNote: true,
          type: 'link',
          properties: { media: 'plugin://org.example.links/tide.mp4' },
        })),
    }))
    const listRelatedNotes = vi.fn((request: { paths: string[]; depth: number }) => ({
      graphs: request.paths.map((path) => ({
        focus: { path, title: 'Root' },
        focusType: 'note',
        relatedNotes: [
          { note: { path: 'Child.md', title: 'Child' }, relation: 2, relatedTo: '' },
          { note: { path: 'Leaf.md', title: 'Leaf' }, relation: 2, relatedTo: 'Child.md' },
        ],
        connections: [],
      })),
    }))
    const api = createAnsweredApi({
      notes: {
        listRelatedNotes,
        listHeadings,
      },
      files: { listFileKinds },
    })
    const search = {
      searchNames: vi.fn(() =>
        Promise.resolve([
          { path: 'One.md', title: 'One', heading: '', line: -1, at: [], type: 'note' as const },
        ]),
      ),
    }
    const pluginNotes = {
      readNote: vi.fn(() => Promise.resolve({ body: 'Prose.' })),
      writeNote: vi.fn(() => Promise.resolve()),
      write: vi.fn(() => Promise.resolve('next')),
    }
    return {
      making,
      listRelatedNotes,
      listHeadings,
      listFileKinds,
      search,
      pluginNotes,
      notes: createVaultNotes({
        notes: api.vault.note,
        getFileKinds: api.vault.file.getFileKinds,
        search,
        making,
        pluginNotes,
      }),
    }
  }

  it('reads and writes the prose of a note through the queue of the plugins', async () => {
    const { notes, pluginNotes } = createNotes()

    expect(await notes.read('Root.md')).toStrictEqual({ body: 'Prose.' })
    const observed = { body: 'Prose.', at: 'first' }
    expect(await notes.write('Root.md', 'More prose.', observed)).toBe('next')

    expect(pluginNotes.readNote).toHaveBeenCalledWith('Root.md')
    expect(pluginNotes.write).toHaveBeenCalledWith('Root.md', 'More prose.', observed)
  })

  it('reads what is around a note and the headings of notes', async () => {
    const { notes, listHeadings } = createNotes()

    expect((await notes.listRelatedNotes(['Root.md']))[0]?.focus).toStrictEqual({
      path: 'Root.md',
      title: 'Root',
    })
    await notes.getHeadings(['Root.md'])
    expect(listHeadings.mock.calls[0]?.[0]).toMatchObject({ paths: ['Root.md'] })
  })

  it('asks the core for depth 1 unless the plugin says 2, and maps the note an item belongs to', async () => {
    const { notes, listRelatedNotes } = createNotes()

    const [one] = await notes.listRelatedNotes(['Root.md'])
    const [two] = await notes.listRelatedNotes(['Root.md'], 2)

    expect(listRelatedNotes.mock.calls.map((call) => call[0].depth)).toEqual([1, 2])
    expect(one?.related.map((item) => item.relatedTo)).toEqual([undefined, 'Child.md'])
    expect(two?.related.map((item) => item.relatedTo)).toEqual([undefined, 'Child.md'])
  })

  it('passes the paths and the depth to the notes api in one request', async () => {
    const { notes, listRelatedNotes } = createNotes()

    const graphs = await notes.listRelatedNotes(['Root.md', 'Other.md'], 2)

    expect(listRelatedNotes).toHaveBeenCalledTimes(1)
    expect(listRelatedNotes.mock.calls[0]?.[0]).toMatchObject({
      paths: ['Root.md', 'Other.md'],
      depth: 2,
    })
    expect(graphs.map((graph) => graph.focus.path)).toEqual(['Root.md', 'Other.md'])
  })

  it('keeps the depth 1 related notes of a path in memory until cleared', async () => {
    const { notes, listRelatedNotes } = createNotes()

    await notes.listRelatedNotes(['Root.md', 'Other.md'], 1)
    const [cachedRoot, loadedThird] = await notes.listRelatedNotes(['Root.md', 'Third.md'], 1)

    expect(listRelatedNotes).toHaveBeenCalledTimes(2)
    expect(listRelatedNotes.mock.calls[1]?.[0]).toMatchObject({
      paths: ['Third.md'],
      depth: 1,
    })
    expect(cachedRoot?.focus.path).toBe('Root.md')
    expect(loadedThird?.focus.path).toBe('Third.md')

    notes.clearRelatedNotes()
    await notes.listRelatedNotes(['Root.md'], 1)
    expect(listRelatedNotes).toHaveBeenCalledTimes(3)
  })

  it('finds notes by name, twenty unless told how many', async () => {
    const { notes, search } = createNotes()

    expect(await notes.findNotes('On')).toStrictEqual([{ path: 'One.md', title: 'One' }])
    await notes.findNotes('On', 5)

    expect(search.searchNames).toHaveBeenNthCalledWith(1, 'On', 20)
    expect(search.searchNames).toHaveBeenNthCalledWith(2, 'On', 5)
  })

  it('makes notes through the note maker, untitled at the root of the vault', async () => {
    const { notes, making } = createNotes()

    expect(await notes.createUntitledNote()).toBe('Untitled.md')
    expect(making.createUntitledNote).toHaveBeenCalledWith('')
  })

  it('reads the frontmatter of a note, and nothing for a path holding no note', async () => {
    const { notes, listFileKinds } = createNotes()

    expect(await notes.readProperties('links/tide.md')).toStrictEqual({
      media: 'plugin://org.example.links/tide.mp4',
    })
    expect(await notes.readProperties('Gone.md')).toStrictEqual({})
    expect(listFileKinds.mock.calls[0]?.[0]).toMatchObject({ paths: ['links/tide.md'] })
  })

  it('makes no path where the note maker made no untitled note', async () => {
    const { notes, making } = createNotes()
    vi.mocked(making.createUntitledNote).mockResolvedValueOnce('')

    expect(await notes.createUntitledNote()).toBe('')
  })
})

describe('createWorkspaceService, createVaultAccess and createWindowService', () => {
  it('wires workspace, vault access and window service from window dependencies', async () => {
    const writers = new Map<string, ReturnType<typeof vi.fn<MessageWriter>>>()
    const getWriter = vi.fn((name: string): MessageWriter => {
      const existing = writers.get(name) ?? vi.fn<MessageWriter>()
      writers.set(name, existing)
      return existing
    })
    const windowTabs = createTestTabs()
    const icon = blank
    const workspace = createWorkspaceService({
      log: { getWriter },
      windowTabs,
      tabOpeners: {
        openFile: vi.fn(() => Promise.reject(new ReportedError('cannot open'))),
        openFileAt: vi.fn(() => Promise.resolve()),
        findFileIcon: (path) => (path === 'a.md' ? icon : undefined),
      },
      tabSync: { syncPathFromTab: () => true },
      window: {
        health: { failure: ref('') },
        opening: { path: ref(''), read: async () => '' },
      },
      getPluginCommandIcon: (id) => (id === 'cmd:1' ? icon : undefined),
    })

    expect(getWriter).toHaveBeenCalledWith('command')
    workspace.messages.showError('bad command')
    expect(writers.get('command')).toHaveBeenCalledWith('bad command', 'error')

    workspace.messages.getWriter('plugin:test')('hello', 'info')
    expect(writers.get('plugin:test')).toHaveBeenCalledWith('hello', 'info')

    expect(workspace.icons.getCommand('cmd:1')).toBe(icon)
    expect(workspace.getNoteInFocus()).toStrictEqual({ path: '', title: '' })
    expect(workspace.window.getToday()).toMatch(/^\d{4}-\d{2}-\d{2}$/)

    workspace.files.open('a.md')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(writers.get('command')).toHaveBeenCalledWith('cannot open', 'error')

    const createUntitledNote = vi.fn(async () => 'New.md')
    const api = createAnsweredApi({})
    const access = createVaultAccess({
      notes: api.vault.note,
      files: api.vault.file,
      downloads: api.vault.downloads,
      relations: { roles: [] } as never,
      search: { searchNames: async () => [] },
      making: { createUntitledNote, createWithTitle: async () => null } as never,
      pluginNotes: {
        readNote: async () => ({ body: '' }),
        writeNote: async () => undefined,
        write: async () => 'next',
      },
      invoking: { getTarget: () => ({}) as never, runCommand: async () => undefined },
    })
    expect(await access.notes.createUntitledNote()).toBe('New.md')

    const writeWindowMessage = vi.fn()
    const windowService = createWindowService({
      frame: {
        isMaximized: ref(false),
        isFullscreen: ref(false),
        minimizeWindow: async () => {
          throw new ReportedError('frame broken')
        },
        toggleMaximizeWindow: async () => undefined,
        closeWindow: async () => undefined,
      },
      writeMessage: writeWindowMessage,
    })
    windowService.minimize()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(writeWindowMessage).toHaveBeenCalledWith('frame broken', 'error')
  })
})
