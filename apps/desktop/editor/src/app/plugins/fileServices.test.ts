/**
 * The files of the vault as a plugin's tabs are given them, and the window's
 * commands over a row.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatErrorMessage } from '@numen/wire'
import { Folder, FolderOpen } from '@lucide/vue'
import { WORDS as words } from '@/shared/words'
import { ANSWER_WORDS, createInvocation, runInvocation } from '@/features/command-palette'
import { running } from '@/entities/artifact'
import { findIcon } from '@/shared/icons'
import { createFileMakers, createPathCommandHost, createVaultFiles } from './fileServices'

vi.mock('@/entities/artifact', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/entities/artifact')>()),
  running: { fetchArtifact: vi.fn(() => Promise.resolve()) },
}))
vi.mock('@/features/command-palette', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/command-palette')>()),
  runInvocation: vi.fn(() => Promise.resolve()),
}))

const target = {
  tab: 't',
  kind: null,
  vault: { id: 'v', name: 'V' },
  made: {},
  isReady: true,
  path: 'target',
  title: '',
  file: '',
  source: null,
}

const setup = () => {
  const writeMessage = vi.fn()
  const openNewFile = vi.fn()
  const createUrl = vi.fn()
  const createFileOfKind = vi.fn()
  const applyPathChanges = vi.fn()
  const createUntitled = vi.fn()
  const commandDeps = vi.fn(() => ({ marker: 'deps' }))
  const createdItem = createFileMakers({
    core: { createUrl },
    tabOpeners: { openNewFile } as never,
    windowTabs: { createFileOfKind },
    notes: { applyPathChanges },
    writeMessage,
  })
  const files = createVaultFiles({
    core: { list: (folder) => Promise.resolve([{ path: `${folder}/a.md` }] as never) },
    made: createdItem,
    making: { createUntitled } as never,
    getTarget: () => target as never,
    commandDeps: commandDeps as never,
  })
  return {
    files,
    writeMessage,
    openNewFile,
    createUrl,
    createFileOfKind,
    applyPathChanges,
    createUntitled,
    commandDeps,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the files of the vault a plugin is handed', () => {
  it('list a folder as the vault lists it', async () => {
    const { files } = setup()

    expect(await files.list('dir')).toStrictEqual([{ path: 'dir/a.md' }])
  })

  it('move a path through the move command, answered in the command words', async () => {
    const { files, commandDeps } = setup()

    await files.move('a.md', 'dir/a.md')

    expect(runInvocation).toHaveBeenCalledWith(
      createInvocation('move', { ...target, path: 'a.md' } as never, 'dir/a.md'),
      commandDeps(),
      ANSWER_WORDS,
    )
  })

  it('make a folder through the createFolder command, answered in the command words', async () => {
    const { files, commandDeps } = setup()

    await files.createFolder('dir')

    expect(runInvocation).toHaveBeenCalledWith(
      createInvocation('createFolder', target as never, 'dir'),
      commandDeps(),
      ANSWER_WORDS,
    )
  })

  it('make a note in a folder and answer at its current position, or no path where none was made', async () => {
    const { files, createUntitled } = setup()
    createUntitled.mockResolvedValueOnce({ path: 'dir/Untitled.md' }).mockResolvedValueOnce(null)

    expect(await files.createNote('dir')).toBe('dir/Untitled.md')
    expect(await files.createNote('dir')).toBe('')
    expect(createUntitled).toHaveBeenCalledWith('dir', [])
  })

  it('make a deck, a stencil and a preset through the kinds that edit them, under the names given', async () => {
    const { files, openNewFile, createFileOfKind } = setup()
    createFileOfKind.mockImplementation((kind: string) =>
      Promise.resolve({ ok: true, value: { path: `dir/${kind}.md` } }),
    )

    expect(await files.createFile('deck', 'dir', 'D')).toBe('dir/deck.md')
    expect(await files.createFile('stencil', 'dir', 'S')).toBe('dir/stencil.md')
    expect(await files.createFile('preset', 'dir', 'P')).toBe('dir/preset.md')

    expect(createFileOfKind.mock.calls).toStrictEqual([
      ['deck', 'dir', 'D'],
      ['stencil', 'dir', 'S'],
      ['preset', 'dir', 'P'],
    ])
    expect(openNewFile).not.toHaveBeenCalled()
  })

  it('make no deck, and say so, where no kind of tab makes one', async () => {
    const { files, writeMessage } = setup()

    expect(await files.createFile('deck', 'dir', 'D')).toBe('')

    expect(writeMessage.mock.calls).toEqual([[words.unavailable, 'error']])
  })

  it('say why the vault rejected a stencil, in the words of the window', async () => {
    const { files, createFileOfKind, writeMessage } = setup()
    createFileOfKind.mockResolvedValue({ ok: false, error: 'occupied' })

    expect(await files.createFile('stencil', 'dir', 'S')).toBe('')

    expect(writeMessage.mock.calls).toEqual([[words.errors.occupied, 'error']])
  })

  it('import several addresses without opening tabs, and one into a tab of its own', async () => {
    const { files, createUrl, openNewFile } = setup()
    createUrl
      .mockResolvedValueOnce({ ok: true, value: { path: 'dir/A.md' } })
      .mockResolvedValueOnce({ ok: true, value: { path: 'dir/B.md' } })
      .mockResolvedValueOnce({ ok: true, value: { path: 'dir/C.md' } })

    expect(await files.importUrls('dir', ['https://a.test', 'https://b.test'])).toEqual([
      'dir/A.md',
      'dir/B.md',
    ])
    expect(openNewFile).not.toHaveBeenCalled()
    expect(await files.importUrls('dir', ['https://c.test'])).toEqual(['dir/C.md'])
    expect(openNewFile).toHaveBeenCalledWith('dir/C.md', '', 'url')
  })
})

describe('an address imported', () => {
  it('fetches what the note points at and notifies the notes its path changed', async () => {
    const { files, createUrl, applyPathChanges } = setup()
    createUrl.mockResolvedValue({ ok: true, value: { path: 'dir/U.md' } })

    expect(await files.importUrl('dir', 'https://example.test')).toBe('dir/U.md')

    expect(createUrl).toHaveBeenCalledWith('https://example.test', 'dir')
    await vi.waitFor(() => expect(applyPathChanges).toHaveBeenCalledWith(['dir/U.md']))
    expect(running.fetchArtifact).toHaveBeenCalledWith('dir/U.md')
  })

  it('says why the fetch failed and does not tell the notes', async () => {
    const { files, createUrl, applyPathChanges, writeMessage } = setup()
    createUrl.mockResolvedValue({ ok: true, value: { path: 'dir/U.md' } })
    const failure = new Error('offline')
    vi.mocked(running.fetchArtifact).mockRejectedValueOnce(failure)

    await files.importUrl('dir', 'https://example.test')

    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith(formatErrorMessage(failure), 'error'),
    )
    expect(applyPathChanges).not.toHaveBeenCalled()
  })

  it('fetches nothing where the vault rejected the address', async () => {
    const { files, createUrl, applyPathChanges, writeMessage } = setup()
    createUrl.mockResolvedValue({ ok: false, error: 'occupied' })

    await files.importUrl('dir', 'https://example.test')

    expect(writeMessage.mock.calls).toEqual([[words.errors.occupied, 'error']])
    expect(running.fetchArtifact).not.toHaveBeenCalled()
    expect(applyPathChanges).not.toHaveBeenCalled()
  })
})

describe('dynamic creator registration', () => {
  it('registers custom file creators and creates files of registered kinds', async () => {
    const writeMessage = vi.fn()
    const openNewFile = vi.fn()
    const makers = createFileMakers({
      tabOpeners: { openNewFile } as never,
      windowTabs: { createFileOfKind: vi.fn() },
      writeMessage,
    })

    makers.registerCreator('kanban', async (title, folder) => ({
      ok: true,
      value: { path: `${folder}/${title}.kanban` },
    }))

    const path = await makers.createFile('kanban', 'boards', 'Sprint')
    expect(path).toBe('boards/Sprint.kanban')
  })
})

describe('the window’s commands over a row', () => {
  const createCommands = () => {
    const runCommand = vi.fn()
    const canRun = vi.fn((run: string) => run !== 'transcribe')
    const makes = { value: new Map([['Lecture.mp3', { transcript: 'done' as const }]]) }
    const host = createPathCommandHost({
      runs: { canRun },
      vaults: { makes },
      getTarget: () => target as never,
      runCommand,
    })
    return { host, runCommand, canRun }
  }
  const entry = (path: string, kind: 'note' | 'recording' | 'book' = 'note', isFolder = false) =>
    ({ path, name: path, isFolder, kind, type: 'note' }) as const

  it('are the file commands over a recording', () => {
    const { host } = createCommands()

    const ids = host.getPathCommands({ entry: entry('Lecture.mp3', 'recording'), isSeveral: false })

    expect(ids.map((one) => [one.id, one.isDisabled])).toStrictEqual([
      ['copy', undefined],
      ['remove', undefined],
    ])
  })

  it('offers removal over several rows and file commands over a document', () => {
    const { host } = createCommands()

    expect(
      host.getPathCommands({ entry: entry('a.md'), isSeveral: true }).map((one) => one.id),
    ).toStrictEqual(['remove'])
    const pdf = host.getPathCommands({ entry: entry('Heat.pdf', 'book'), isSeveral: false })
    expect(pdf.map((one) => one.id)).toStrictEqual(['copy', 'remove'])
  })

  it('run over the first path, with the rest as the others and what the vault made', () => {
    const { host, runCommand } = createCommands()

    host.runPathCommand('remove', {
      paths: ['Lecture.mp3', 'b.md'],
      name: 'Lecture',
      source: 'recording',
    })
    host.runPathCommand('remove', { paths: [], name: 'A', source: 'note' })

    expect(runCommand).toHaveBeenNthCalledWith(1, 'remove', {
      ...target,
      path: 'Lecture.mp3',
      title: 'Lecture',
      file: 'Lecture.mp3',
      source: 'recording',
      made: { transcript: 'done' },
      others: ['b.md'],
    })
    expect(runCommand.mock.calls.at(1)?.at(1)).toMatchObject({
      path: '',
      file: '',
      made: {},
      others: [],
    })
  })

  it('are drawn with the window’s icons, and an entry by what it is', () => {
    const { host } = createCommands()

    expect(host.getCommandIcon('rename')).toBe(findIcon('rename'))
    expect(host.getCommandIcon('nothing at all')).toBeNull()
    expect(host.getEntryIcon(entry('physics', 'note', true), false)).toBe(Folder)
    expect(host.getEntryIcon(entry('physics', 'note', true), true)).toBe(FolderOpen)
  })
})
