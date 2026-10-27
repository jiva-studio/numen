/**
 * The files of the vault as a plugin's tabs are given them, and the window's
 * commands over a row.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Folder, FolderOpen } from '@lucide/vue'
import { ANSWER_WORDS, createInvocation, runInvocation } from '@/features/command-palette'
import { findIcon } from '@/shared/icons'
import { createFileMakers, createPathCommandHost, createVaultFiles } from './fileServices'

vi.mock('@/features/command-palette', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/command-palette')>()),
  runInvocation: vi.fn(() => Promise.resolve()),
}))

const target = {
  tab: 't',
  kind: null,
  vault: { id: 'v', name: 'V' },
  isReady: true,
  path: 'target',
  title: '',
  file: '',
  source: null,
}

const setup = () => {
  const createUntitled = vi.fn()
  const commandDeps = vi.fn(() => ({ marker: 'deps' }))
  const files = createVaultFiles({
    core: { list: (folder) => Promise.resolve([{ path: `${folder}/a.md` }] as never) },
    making: { createUntitled } as never,
    getTarget: () => target as never,
    commandDeps: commandDeps as never,
  })
  return { files, createUntitled, commandDeps }
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
    const host = createPathCommandHost({
      getTarget: () => target as never,
      runCommand,
    })
    return { host, runCommand }
  }
  const entry = (path: string, isNote = true, isFolder = false) =>
    ({ path, name: path, isFolder, isNote, type: 'note' }) as const

  it('are the file commands over a file that is no note', () => {
    const { host } = createCommands()

    const ids = host.getPathCommands({ entry: entry('Lecture.mp3', false), isSeveral: false })

    expect(ids.map((one) => [one.id, one.isDisabled])).toStrictEqual([
      ['copy', undefined],
      ['remove', undefined],
    ])
  })

  it('offers removal over several rows and file commands over a file', () => {
    const { host } = createCommands()

    expect(
      host.getPathCommands({ entry: entry('a.md'), isSeveral: true }).map((one) => one.id),
    ).toStrictEqual(['remove'])
    const pdf = host.getPathCommands({ entry: entry('Heat.pdf', false), isSeveral: false })
    expect(pdf.map((one) => one.id)).toStrictEqual(['copy', 'remove'])
  })

  it('run over the first path, with the rest as the others', () => {
    const { host, runCommand } = createCommands()

    host.runPathCommand('remove', {
      paths: ['Lecture.mp3', 'b.md'],
      name: 'Lecture',
      isNote: false,
      isFolder: false,
    })
    host.runPathCommand('remove', { paths: [], name: 'A', isNote: true, isFolder: false })

    expect(runCommand).toHaveBeenNthCalledWith(1, 'remove', {
      ...target,
      path: 'Lecture.mp3',
      title: 'Lecture',
      file: 'Lecture.mp3',
      others: ['b.md'],
    })
    expect(runCommand.mock.calls.at(1)?.at(1)).toMatchObject({
      path: '',
      file: '',
      others: [],
    })
  })

  it('are drawn with the window’s icons, and an entry by what it is', () => {
    const { host } = createCommands()

    expect(host.getCommandIcon('rename')).toBe(findIcon('rename'))
    expect(host.getCommandIcon('nothing at all')).toBeNull()
    expect(host.getEntryIcon(entry('physics', true, true), false)).toBe(Folder)
    expect(host.getEntryIcon(entry('physics', true, true), true)).toBe(FolderOpen)
  })
})
