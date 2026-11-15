/**
 * What a plugin may do through the window, the notes and the files it is
 * handed: the reads need `vault:read`, the writes need `vault:write`.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  asValue,
  isUnsavedBuffersError,
  UnsavedBuffersError,
  type BufferHost,
  type HostServices,
  type VaultFiles,
  type VaultNotes,
  type VaultRelations,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createMockCommandsHost } from '@numen/plugin-sdk/testing'
import { createPluginParts } from './context'
import { disposeSubscriptions } from './guards/subscriptions'
import type { PluginContextHost } from './types'
import { createTestContextHost } from '../testing/contextHost'

const createEmptyHost = (): PluginContextHost => createTestContextHost()

const LISTED_COMMANDS = [
  'app:note.trash',
  'app:note.createChild',
  'app:note.rename',
  'md.numen.engines.transcription:transcript.create',
]

const COMMAND_PERMISSIONS: Readonly<Record<string, string>> = {
  'md.numen.other:vault.archive': 'vault:manage',
}

const createServices = () => {
  const workspace = {
    files: { open: vi.fn() },
    messages: { showError: vi.fn() },
  } as unknown as WorkspaceHost
  const notes = {
    listRelatedNotes: vi.fn(() => Promise.resolve({ path: 'Notes/Example.md' })),
    getHeadings: vi.fn(() => Promise.resolve(new Map())),
    findNotes: vi.fn(() => Promise.resolve([])),
    read: vi.fn(() => Promise.resolve({ body: 'Example prose.' })),
    readProperties: vi.fn(() => Promise.resolve({ title: 'Example' })),
    createUntitledNote: vi.fn(() => Promise.resolve('Untitled.md')),
    write: vi.fn(() => Promise.resolve()),
  } as unknown as VaultNotes
  const relations = {
    roles: ['parent', 'child', 'jump'],
    getConnection: vi.fn(() => Promise.resolve(asValue({}))),
    saveConnection: vi.fn(() => Promise.resolve(asValue(undefined))),
    removeSide: vi.fn(() => Promise.resolve(asValue(undefined))),
    removeConnection: vi.fn(() => Promise.resolve(asValue(undefined))),
    join: vi.fn(() => Promise.resolve(asValue(undefined))),
    createRelated: vi.fn(() => Promise.resolve(asValue('a.md'))),
    listBacklinks: vi.fn(() => Promise.resolve([])),
    resolveLinks: vi.fn(() => Promise.resolve([])),
  } as unknown as VaultRelations
  const files = {
    list: vi.fn(() => Promise.resolve([])),
    move: vi.fn(() => Promise.resolve()),
    createFolder: vi.fn(() => Promise.resolve()),
    createNote: vi.fn(() => Promise.resolve('a.md')),
  } as unknown as VaultFiles
  const buffers = {
    track: vi.fn(() => ({ dispose: vi.fn() })),
    addFlushHandler: vi.fn(() => ({ dispose: vi.fn() })),
  } as unknown as BufferHost
  const commands = createMockCommandsHost()
  const host = createTestContextHost({
    vault: {
      getBuffers: () => buffers,
      getNotes: () => notes,
      getRelations: () => relations,
      getFiles: () => files,
    },
    ui: {
      getWorkspace: () => workspace,
      getCommands: () => commands,
      isCommandListed: (command) =>
        LISTED_COMMANDS.includes(command) || command in COMMAND_PERMISSIONS,
      getCommandPermission: (command) => COMMAND_PERMISSIONS[command],
    },
  })
  return { host, workspace, notes, relations, files, buffers, commands }
}

const createContext = (permissions: readonly string[]) => {
  const services = createServices()
  const context = createPluginParts('md.numen.sample', services.host, {
    id: 'md.numen.sample',
    name: 'Sample',
    version: '1.0.0',
    permissions,
  })
  return { ...services, context }
}

type Call = (services: HostServices) => unknown

const NOTE_WRITES: Record<string, Call> = {
  createUntitledNote: (c) => c.vault.notes!.createUntitledNote(),
  write: (c) => c.vault.notes!.write('a.md', 'Prose.'),
}

const NOTE_READS: Record<string, Call> = {
  listRelatedNotes: (c) => c.vault.notes!.listRelatedNotes('Notes/Example.md'),
  getHeadings: (c) => c.vault.notes!.getHeadings(['Notes/Example.md']),
  findNotes: (c) => c.vault.notes!.findNotes('Example', 5),
  read: (c) => c.vault.notes!.read('Notes/Example.md'),
  readProperties: (c) => c.vault.notes!.readProperties('Notes/Example.md'),
}

const RELATION_WRITES: Record<string, Call> = {
  saveConnection: (c) =>
    c.vault.relations!.saveConnection({
      first: 'a.md',
      second: 'b.md',
      role: 'child',
      type: '',
      firstSide: { label: '', marker: 'arrow', reason: '' },
      secondSide: null,
    }),
  removeSide: (c) => c.vault.relations!.removeSide('a.md', 'b.md'),
  removeConnection: (c) => c.vault.relations!.removeConnection('a.md', 'b.md'),
  join: (c) => c.vault.relations!.join('a.md', 'b.md', { role: 'child' }),
  createRelated: (c) => c.vault.relations!.createRelated({ from: 'a.md', role: 'child' }),
}

const RELATION_READS: Record<string, Call> = {
  getConnection: (c) => c.vault.relations!.getConnection('a.md', 'b.md'),
  listBacklinks: (c) => c.vault.relations!.listBacklinks('a.md'),
  resolveLinks: (c) => c.vault.relations!.resolveLinks('See [[Example]].'),
}

describe('the relationships a plugin writes', () => {
  it.each(Object.entries(RELATION_WRITES))(
    'fail through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for relations.${name}`,
      )
    },
  )

  it.each(Object.entries(RELATION_WRITES))(
    'are written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context.host))).resolves.not.toThrow()
    },
  )

  it.each(Object.entries(RELATION_READS))(
    'fail through %s without vault:read',
    async (name, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:read" for relations.${name}`,
      )
    },
  )

  it.each(Object.entries(RELATION_READS))(
    'are read through %s with vault:read',
    async (_, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context.host))).resolves.not.toThrow()
    },
  )

  it('pass the roles the window offers', () => {
    const { context } = createContext([])

    expect(context.host.vault.relations!.roles).toStrictEqual(['parent', 'child', 'jump'])
  })
})

const FILE_WRITES: Record<string, Call> = {
  move: (c) => c.vault.files!.move('a.md', 'b.md'),
  createFolder: (c) => c.vault.files!.createFolder('dir'),
  createNote: (c) => c.vault.files!.createNote('dir'),
}

describe('a note a plugin reads and writes', () => {
  it.each(Object.entries(NOTE_WRITES))(
    'fails through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for notes.${name}`,
      )
    },
  )

  it.each(Object.entries(NOTE_WRITES))(
    'is written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context.host))).resolves.not.toThrow()
    },
  )

  it.each(Object.entries(NOTE_READS))('fails through %s without vault:read', async (name, call) => {
    const { context, notes } = createContext(['vault:write'])

    await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
      `Plugin "md.numen.sample" lacks required permission "vault:read" for notes.${name}`,
    )
    expect(notes[name as keyof VaultNotes]).not.toHaveBeenCalled()
  })

  it.each(Object.entries(NOTE_READS))('fails through %s with no permission', async (name, call) => {
    const { context } = createContext([])

    await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
      `Plugin "md.numen.sample" lacks required permission "vault:read" for notes.${name}`,
    )
  })

  it.each(Object.entries(NOTE_READS))('is read through %s with vault:read', async (name, call) => {
    const { context, notes } = createContext(['vault:read'])

    await call(context.host)

    expect(notes[name as keyof VaultNotes]).toHaveBeenCalledOnce()
  })

  it('a related-notes read sends its depth to the editor service unchanged', async () => {
    const { context, notes } = createContext(['vault:read'])

    await context.host.vault.notes!.listRelatedNotes('Notes/Example.md', 2)
    await context.host.vault.notes!.listRelatedNotes('Notes/Example.md')

    expect(notes.listRelatedNotes).toHaveBeenNthCalledWith(1, 'Notes/Example.md', 2)
    expect(notes.listRelatedNotes).toHaveBeenNthCalledWith(2, 'Notes/Example.md')
  })

  it('is read with vault:read and without vault:write', async () => {
    const { context, notes } = createContext(['vault:read'])

    const note = await context.host.vault.notes!.read('Notes/Example.md')

    expect(note).toStrictEqual({ body: 'Example prose.' })
    expect(notes.read).toHaveBeenCalledWith('Notes/Example.md')
  })

  it('is written and not read with vault:write and without vault:read', async () => {
    const { context, notes } = createContext(['vault:write'])

    await context.host.vault.notes!.write('Notes/Example.md', 'New prose.')

    expect(notes.write).toHaveBeenCalledWith('Notes/Example.md', 'New prose.')
    await expect(context.host.vault.notes!.read('Notes/Example.md')).rejects.toThrowError(
      'Plugin "md.numen.sample" lacks required permission "vault:read" for notes.read',
    )
  })
})

const BUFFER_WRITES: Record<string, Call> = {
  track: (c) => c.vault.buffers!.track({} as never),
  addFlushHandler: (c) => c.vault.buffers!.addFlushHandler(vi.fn(() => Promise.resolve())),
}

describe('the open files a plugin passes the window', () => {
  it.each(Object.entries(BUFFER_WRITES))(
    'fail through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for buffers.${name}`,
      )
    },
  )

  it.each(Object.entries(BUFFER_WRITES))(
    'are taken through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context.host))).resolves.not.toThrow()
    },
  )

  it('are passed to the window the plugin runs in, and to none where the window made none', () => {
    const { context, buffers } = createContext(['vault:write'])
    context.host.vault.buffers!.track({} as never)

    expect(buffers.track).toHaveBeenCalledOnce()
    expect(
      createPluginParts('md.numen.sample', createEmptyHost(), {
        permissions: ['vault:write'],
      }),
    ).not.toHaveProperty('buffers')
  })
})

describe('the text a plugin could not write at quit', () => {
  const unsaved = [{ name: 'Decks.md', keep: vi.fn(), take: vi.fn() }]

  it('is reported to the window under the id of the plugin, whichever way it flushes', async () => {
    const { context, buffers } = createContext(['vault:write'])
    const flush = () => Promise.reject(new UnsavedBuffersError(unsaved))
    context.host.vault.buffers!.track({ flush })
    context.host.vault.buffers!.addFlushHandler(flush)

    const handed = [
      vi.mocked(buffers.track).mock.calls[0]![0].flush,
      vi.mocked(buffers.addFlushHandler).mock.calls[0]![0],
    ]
    for (const hand of handed) {
      const error: unknown = await hand().catch((caught: unknown) => caught)
      expect(isUnsavedBuffersError(error) && error.pluginId).toBe('md.numen.sample')
      expect(isUnsavedBuffersError(error) && error.unsaved).toEqual(unsaved)
    }
  })

  it('leaves any other failure of a flush as it was', async () => {
    const { context, buffers } = createContext(['vault:write'])
    const failure = new Error('disk gone')
    context.host.vault.buffers!.track({ flush: () => Promise.reject(failure) })

    const handed = vi.mocked(buffers.track).mock.calls[0]![0]

    await expect(handed.flush()).rejects.toBe(failure)
  })
})

describe('the registrations of a plugin in the open files', () => {
  it('are released when the plugin is disposed', () => {
    const { context, buffers } = createContext(['vault:write'])
    const released = vi.fn()
    vi.mocked(buffers.track).mockReturnValue({ dispose: released })
    vi.mocked(buffers.addFlushHandler).mockReturnValue({ dispose: released })

    context.host.vault.buffers!.track({} as never)
    context.host.vault.buffers!.addFlushHandler(vi.fn(() => Promise.resolve()))
    expect(context.plugin.identity.subscriptions).toHaveLength(2)
    disposeSubscriptions(context.plugin.identity.subscriptions)

    expect(released).toHaveBeenCalledTimes(2)
  })
})

describe('a file a plugin lists and writes', () => {
  it.each(Object.entries(FILE_WRITES))(
    'fails through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context.host))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for files.${name}`,
      )
    },
  )

  it.each(Object.entries(FILE_WRITES))(
    'is written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context.host))).resolves.not.toThrow()
    },
  )

  it('is listed with vault:read and without vault:write', async () => {
    const { context, files } = createContext(['vault:read'])

    await context.host.vault.files!.list('Notes')

    expect(files.list).toHaveBeenCalledWith('Notes')
  })

  it.each([
    ['vault:write', ['vault:write']],
    ['no permission', []],
  ])('fails to be listed with %s', async (_, permissions) => {
    const { context, files } = createContext(permissions)

    await expect(context.host.vault.files!.list('Notes')).rejects.toThrowError(
      'Plugin "md.numen.sample" lacks required permission "vault:read" for files.list',
    )
    expect(files.list).not.toHaveBeenCalled()
  })
})

describe('a command a plugin runs over the vault', () => {
  const target = { path: 'a.md', title: 'A', isFolder: false, others: [], tab: '' }
  const WRITES = LISTED_COMMANDS

  it.each(WRITES)('runs %s through the commands of the window with vault:write', async (id) => {
    const { context, commands } = createContext(['vault:write'])

    await context.host.ui.commands!.run(id, target)

    expect(commands.getRuns()).toStrictEqual([{ command: id, target, itemId: undefined }])
  })

  it('runs a command of its own that no slot lists', async () => {
    const { context, commands } = createContext(['vault:write'])

    await context.host.ui.commands!.run('md.numen.sample:deck.create', target)

    expect(commands.getRuns()).toHaveLength(1)
  })

  it.each(['app:vault.create', 'app:settings.open'])(
    'runs the application command %s that no slot lists',
    async (id) => {
      const { context, commands } = createContext(['vault:write'])

      await context.host.ui.commands!.run(id)

      expect(commands.getRuns()).toHaveLength(1)
    },
  )

  it.each([
    'app:vault.delete',
    'md.numen.other:deck.remove',
    'md.numen.sample.evil:deck.remove',
    'x:md.numen.sample:deck.remove',
    'deck.remove',
    'md.numen.sample',
  ])('rejects %s, a command it does not own that no slot lists', async (id) => {
    const { context, commands } = createContext(['vault:write'])

    await expect(context.host.ui.commands!.run(id, target)).rejects.toThrowError(
      /does not own and no slot lists/,
    )
    expect(commands.getRuns()).toStrictEqual([])
  })

  it('asks the permission before the command', async () => {
    const { context } = createContext([])

    await expect(context.host.ui.commands!.run('app:note.trash', target)).rejects.toThrowError(
      'lacks required permission "vault:write" for commands.run',
    )
  })

  it('checks the permission a command declares before running it', async () => {
    const withoutManage = createContext(['vault:write'])
    await expect(
      withoutManage.context.host.ui.commands!.run('md.numen.other:vault.archive', target),
    ).rejects.toThrowError('lacks required permission "vault:manage" for commands.run')
    expect(withoutManage.commands.getRuns()).toStrictEqual([])

    const withManage = createContext(['vault:write', 'vault:manage'])
    await withManage.context.host.ui.commands!.run('md.numen.other:vault.archive', target)
    expect(withManage.commands.getRuns()).toStrictEqual([
      { command: 'md.numen.other:vault.archive', target, itemId: undefined },
    ])
  })

  it('reads the commands of the window and their items with no permission', () => {
    const { context, commands } = createContext([])
    const handler = vi.fn()
    context.host.ui.commands!.watch(handler)

    commands.setIds(['app:vault.create'])

    expect(context.host.ui.commands!.getIds()).toStrictEqual(['app:vault.create'])
    expect(handler).toHaveBeenLastCalledWith(['app:vault.create'])
  })

  it('opens a file with vault:read and shows an error without a permission', () => {
    const { context, workspace } = createContext(['vault:read'])

    context.host.ui.workspace!.files.open('a.md')
    context.host.ui.workspace!.messages.showError('no')

    expect(workspace.files.open).toHaveBeenCalledWith('a.md')
    expect(workspace.messages.showError).toHaveBeenCalledWith('no')
  })
})
