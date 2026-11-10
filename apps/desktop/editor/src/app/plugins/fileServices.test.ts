/**
 * The files of the vault as a plugin's tabs are given them, and the icons the window draws
 * commands and entries with.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { Folder, FolderOpen } from '@lucide/vue'
import { createInvocation, runInvocation } from '@/features/command-palette'
import { COMMAND, findIcon } from '@/entities/command'
import { createFileMakers, createIconHost, createVaultFiles } from './fileServices'
import type * as CommandPaletteModule from '@/features/command-palette'

vi.mock('@/features/command-palette', async (importOriginal) => ({
  ...(await importOriginal<typeof CommandPaletteModule>()),
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
    making: { createUntitled } as never,
    getTarget: () => target as never,
    commandDeps: commandDeps as never,
  })
  return { files, createUntitled, commandDeps }
}

let restore: () => void

beforeEach(() => {
  restore = installCoreAnswers({
    files: {
      listFiles: (request) => ({
        entries: [
          { path: `${request.path}/a.md`, name: 'a.md', isFolder: false, isNote: true, type: '' },
        ],
      }),
    },
  })
  vi.clearAllMocks()
})

afterEach(() => {
  restore()
})

describe('the files of the vault a plugin is handed', () => {
  it('list a folder as the vault lists it', async () => {
    const { files } = setup()

    expect(await files.list('dir')).toStrictEqual([
      { path: 'dir/a.md', name: 'a.md', isFolder: false, isNote: true, type: 'note' },
    ])
  })

  it('move a path through the move command, answered in the command words', async () => {
    const { files, commandDeps } = setup()

    await files.move('a.md', 'dir/a.md')

    expect(runInvocation).toHaveBeenCalledWith(
      createInvocation('app:file.move', { ...target, path: 'a.md' } as never, 'dir/a.md'),
      commandDeps(),
    )
  })

  it('make a folder through the createFolder command, answered in the command words', async () => {
    const { files, commandDeps } = setup()

    await files.createFolder('dir')

    expect(runInvocation).toHaveBeenCalledWith(
      createInvocation('app:folder.create', target as never, 'dir'),
      commandDeps(),
    )
  })

  it('make a note in a folder and answer at its current position, or no path where none was made', async () => {
    const { files, createUntitled } = setup()
    createUntitled.mockResolvedValueOnce('dir/Untitled.md').mockResolvedValueOnce('')

    expect(await files.createNote('dir')).toBe('dir/Untitled.md')
    expect(await files.createNote('dir')).toBe('')
    expect(createUntitled).toHaveBeenCalledWith('dir')
  })
})

describe('dynamic creator registration', () => {
  it('registers custom file creators, falls back to windowTabs, and handles errors', async () => {
    const writeMessage = vi.fn()
    const openFile = vi.fn()
    const createFileOfKind = vi.fn()
    const makers = createFileMakers({
      tabOpeners: { openFile } as never,
      windowTabs: { groups: { createFileOfKind } },
      writeMessage,
      creators: {
        preset: async (title, folder) => ({
          ok: true,
          value: { path: `${folder}/${title}.preset` },
        }),
      },
    })

    expect(await makers.createFile('preset', 'dir', 'One')).toBe('dir/One.preset')

    const disposeFirst = makers.registerCreator('kanban', async (title, folder) => ({
      ok: true,
      value: { path: `${folder}/${title}.kanban` },
    }))
    const disposeSecond = makers.registerCreator('kanban', async (title, folder) => ({
      ok: true,
      value: { path: `${folder}/${title}.kanban2` },
    }))
    disposeFirst()
    expect(await makers.createFile('kanban', 'boards', 'Sprint')).toBe('boards/Sprint.kanban2')

    disposeSecond()
    createFileOfKind.mockResolvedValueOnce(undefined)
    expect(await makers.createFile('kanban', 'boards', 'Sprint')).toBe('')
    expect(writeMessage).toHaveBeenCalledTimes(1)
    expect(writeMessage).toHaveBeenCalledWith(
      'Unavailable: the plugin is disabled or stopped.',
      'error',
    )

    writeMessage.mockClear()
    createFileOfKind.mockResolvedValueOnce({ ok: false, error: 'alreadyExists' })
    expect(await makers.createFile('kanban', 'boards', 'Sprint')).toBe('')
    expect(writeMessage).toHaveBeenCalledWith('a file with this name exists', 'error')

    createFileOfKind.mockResolvedValueOnce({ ok: true, value: { path: 'boards/Sprint.kb' } })
    expect(await makers.createFile('kanban', 'boards', 'Sprint')).toBe('boards/Sprint.kb')
  })
})

describe('the icons the window draws with', () => {
  const entry = (path: string, isNote = true, isFolder = false) =>
    ({ path, name: path, isFolder, isNote, type: 'note' }) as const

  it('are all the host offers: no commands over a row', () => {
    expect(Object.keys(createIconHost({})).sort()).toStrictEqual(['getCommandIcon', 'getEntryIcon'])
  })

  it('draw a command with the icon the plugin that offers it gives it', () => {
    const icon = { name: 'PluginIcon' }
    const host = createIconHost({
      getPluginCommandIcon: (id) => (id === 'md.numen.x:a.b' ? icon : undefined),
    })

    expect(host.getCommandIcon('md.numen.x:a.b')).toBe(icon)
    expect(host.getCommandIcon('md.numen.y:a.b')).not.toBe(icon)
  })

  it('draw a command with the window’s icon, and an entry by what it is', () => {
    const host = createIconHost({})

    expect(host.getCommandIcon(COMMAND.renameNote)).toBe(findIcon(COMMAND.renameNote))
    expect(host.getCommandIcon('nothing at all')).toBeNull()
    expect(host.getEntryIcon(entry('physics', true, true), false)).toBe(Folder)
    expect(host.getEntryIcon(entry('physics', true, true), true)).toBe(FolderOpen)
  })
})
