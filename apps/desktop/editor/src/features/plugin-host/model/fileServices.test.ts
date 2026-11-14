/**
 * The files of the vault as a plugin's tabs are given them, and the icons the window draws
 * commands and entries with.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { Folder, FolderOpen } from '@lucide/vue'
import { COMMAND, findIcon } from '@/shared/model/command'
import { createIconHost, createVaultFiles } from './fileServices'

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

const api = createAnsweredApi({
  files: {
    listFiles: (request) => ({
      entries: [
        { path: `${request.path}/a.md`, name: 'a.md', isFolder: false, isNote: true, type: '' },
      ],
    }),
  },
})

const setup = () => {
  const createUntitledNote = vi.fn()
  const runCommand = vi.fn(() => Promise.resolve())
  const files = createVaultFiles({
    listFiles: api.file.listFiles,
    making: { createUntitledNote } as never,
    getTarget: () => target as never,
    runCommand,
  })
  return { files, createUntitledNote, runCommand }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the files of the vault a plugin is handed', () => {
  it('list a folder as the vault lists it', async () => {
    const { files } = setup()

    expect(await files.list('dir')).toStrictEqual([
      { path: 'dir/a.md', name: 'a.md', isFolder: false, isNote: true, type: 'note' },
    ])
  })

  it('move a path through the move command, answered in the command words', async () => {
    const { files, runCommand } = setup()

    await files.move('a.md', 'dir/a.md')

    expect(runCommand).toHaveBeenCalledWith(
      'app:file.move',
      { ...target, path: 'a.md' },
      'dir/a.md',
    )
  })

  it('make a folder through the createFolder command, answered in the command words', async () => {
    const { files, runCommand } = setup()

    await files.createFolder('dir')

    expect(runCommand).toHaveBeenCalledWith('app:folder.create', target, 'dir')
  })

  it('make a note in a folder and answer at its current position, or no path where none was made', async () => {
    const { files, createUntitledNote } = setup()
    createUntitledNote.mockResolvedValueOnce('dir/Untitled.md').mockResolvedValueOnce('')

    expect(await files.createNote('dir')).toBe('dir/Untitled.md')
    expect(await files.createNote('dir')).toBe('')
    expect(createUntitledNote).toHaveBeenCalledWith('dir')
  })
})

describe('the icons the window draws with', () => {
  const entry = (path: string, isNote = true, isFolder = false) =>
    ({ path, name: path, isFolder, isNote, type: 'note' }) as const

  it('are all the host offers: no commands over a row', () => {
    expect(Object.keys(createIconHost({})).sort()).toStrictEqual(['getCommand', 'getEntry'])
  })

  it('draw a command with the icon the plugin that offers it gives it', () => {
    const icon = { name: 'PluginIcon' }
    const host = createIconHost({
      getPluginCommandIcon: (id) => (id === 'md.numen.x:a.b' ? icon : undefined),
    })

    expect(host.getCommand('md.numen.x:a.b')).toBe(icon)
    expect(host.getCommand('md.numen.y:a.b')).not.toBe(icon)
  })

  it('draw a command with the window’s icon, and an entry by what it is', () => {
    const host = createIconHost({})

    expect(host.getCommand(COMMAND.renameNote)).toBe(findIcon(COMMAND.renameNote))
    expect(host.getCommand('nothing at all')).toBeNull()
    expect(host.getEntry(entry('physics', true, true), false)).toBe(Folder)
    expect(host.getEntry(entry('physics', true, true), true)).toBe(FolderOpen)
  })
})
