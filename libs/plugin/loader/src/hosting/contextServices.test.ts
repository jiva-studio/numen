/**
 * What a plugin may do through the window, the notes and the files it is
 * handed: the reads need `vault:read`, the writes need `vault:write`.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  asValue,
  type AgentHost,
  type BufferHost,
  type PluginContext,
  type VaultFiles,
  type VaultNotes,
  type VaultRelations,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createMockCommandsHost } from '@numen/plugin-sdk/testing'
import { createPluginContext } from './context'
import type { PluginContextHost } from './types'

const createServices = () => {
  const workspace = {
    openFile: vi.fn(),
    showError: vi.fn(),
  } as unknown as WorkspaceHost
  const notes = {
    getNeighbourhood: vi.fn(() => Promise.resolve({ path: 'Notes/Example.md' })),
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
  } as unknown as VaultRelations
  const files = {
    list: vi.fn(() => Promise.resolve([])),
    move: vi.fn(() => Promise.resolve()),
    createFolder: vi.fn(() => Promise.resolve()),
    createNote: vi.fn(() => Promise.resolve('a.md')),
  } as unknown as VaultFiles
  const agent = {
    ask: vi.fn(),
    finish: vi.fn(() => Promise.resolve()),
    resolveLinks: vi.fn(() => Promise.resolve([])),
    getUnreachable: vi.fn(() => ''),
    getSetting: vi.fn(),
    writeSetting: vi.fn(() => Promise.resolve()),
    getModelsAt: vi.fn(() => []),
  } as unknown as AgentHost
  const buffers = {
    track: vi.fn(() => ({ dispose: vi.fn() })),
    addFlushHandler: vi.fn(() => ({ dispose: vi.fn() })),
  } as unknown as BufferHost
  const commands = createMockCommandsHost()
  const host = {
    getAgent: () => agent,
    getBuffers: () => buffers,
    getWorkspace: () => workspace,
    getNotes: () => notes,
    getRelations: () => relations,
    getFiles: () => files,
    getCommands: () => commands,
  } as unknown as PluginContextHost
  return { host, workspace, notes, relations, files, agent, buffers, commands }
}

const createContext = (permissions: readonly string[]) => {
  const services = createServices()
  const context = createPluginContext('md.numen.sample', services.host, {
    id: 'md.numen.sample',
    name: 'Sample',
    version: '1.0.0',
    permissions,
  })
  return { ...services, context }
}

type Call = (context: PluginContext) => unknown

const NOTE_WRITES: Record<string, Call> = {
  createUntitledNote: (c) => c.notes!.createUntitledNote(),
  write: (c) => c.notes!.write('a.md', 'Prose.'),
}

const NOTE_READS: Record<string, Call> = {
  getNeighbourhood: (c) => c.notes!.getNeighbourhood('Notes/Example.md'),
  getHeadings: (c) => c.notes!.getHeadings(['Notes/Example.md']),
  findNotes: (c) => c.notes!.findNotes('Example', 5),
  read: (c) => c.notes!.read('Notes/Example.md'),
  readProperties: (c) => c.notes!.readProperties('Notes/Example.md'),
}

const RELATION_WRITES: Record<string, Call> = {
  saveConnection: (c) =>
    c.relations!.saveConnection({
      first: 'a.md',
      second: 'b.md',
      role: 'child',
      type: '',
      firstSide: { label: '', marker: 'arrow', reason: '' },
      secondSide: null,
    }),
  removeSide: (c) => c.relations!.removeSide('a.md', 'b.md'),
  removeConnection: (c) => c.relations!.removeConnection('a.md', 'b.md'),
  join: (c) => c.relations!.join('a.md', 'b.md', { role: 'child' }),
  createRelated: (c) => c.relations!.createRelated({ from: 'a.md', role: 'child' }),
}

const RELATION_READS: Record<string, Call> = {
  getConnection: (c) => c.relations!.getConnection('a.md', 'b.md'),
  listBacklinks: (c) => c.relations!.listBacklinks('a.md'),
}

describe('the relationships a plugin writes', () => {
  it.each(Object.entries(RELATION_WRITES))(
    'fail through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for relations.${name}`,
      )
    },
  )

  it.each(Object.entries(RELATION_WRITES))(
    'are written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it.each(Object.entries(RELATION_READS))(
    'fail through %s without vault:read',
    async (name, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:read" for relations.${name}`,
      )
    },
  )

  it.each(Object.entries(RELATION_READS))(
    'are read through %s with vault:read',
    async (_, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it('pass the roles the window offers', () => {
    const { context } = createContext([])

    expect(context.vault.relations!.roles).toStrictEqual(['parent', 'child', 'jump'])
  })
})

const FILE_WRITES: Record<string, Call> = {
  move: (c) => c.files!.move('a.md', 'b.md'),
  createFolder: (c) => c.files!.createFolder('dir'),
  createNote: (c) => c.files!.createNote('dir'),
}

describe('a note a plugin reads and writes', () => {
  it.each(Object.entries(NOTE_WRITES))(
    'fails through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for notes.${name}`,
      )
    },
  )

  it.each(Object.entries(NOTE_WRITES))(
    'is written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it.each(Object.entries(NOTE_READS))('fails through %s without vault:read', async (name, call) => {
    const { context, notes } = createContext(['vault:write'])

    await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
      `Plugin "md.numen.sample" lacks required permission "vault:read" for notes.${name}`,
    )
    expect(notes[name as keyof VaultNotes]).not.toHaveBeenCalled()
  })

  it.each(Object.entries(NOTE_READS))('fails through %s with no permission', async (name, call) => {
    const { context } = createContext([])

    await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
      `Plugin "md.numen.sample" lacks required permission "vault:read" for notes.${name}`,
    )
  })

  it.each(Object.entries(NOTE_READS))('is read through %s with vault:read', async (name, call) => {
    const { context, notes } = createContext(['vault:read'])

    await call(context)

    expect(notes[name as keyof VaultNotes]).toHaveBeenCalledOnce()
  })

  it('is read with vault:read and without vault:write', async () => {
    const { context, notes } = createContext(['vault:read'])

    const note = await context.vault.notes!.read('Notes/Example.md')

    expect(note).toStrictEqual({ body: 'Example prose.' })
    expect(notes.read).toHaveBeenCalledWith('Notes/Example.md')
  })

  it('is written and not read with vault:write and without vault:read', async () => {
    const { context, notes } = createContext(['vault:write'])

    await context.vault.notes!.write('Notes/Example.md', 'New prose.')

    expect(notes.write).toHaveBeenCalledWith('Notes/Example.md', 'New prose.')
    await expect(context.vault.notes!.read('Notes/Example.md')).rejects.toThrowError(
      'Plugin "md.numen.sample" lacks required permission "vault:read" for notes.read',
    )
  })
})

const BUFFER_WRITES: Record<string, Call> = {
  track: (c) => c.buffers!.track({} as never),
  addFlushHandler: (c) => c.buffers!.addFlushHandler(vi.fn(() => Promise.resolve())),
}

describe('the open files a plugin passes the window', () => {
  it.each(Object.entries(BUFFER_WRITES))(
    'fail through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for buffers.${name}`,
      )
    },
  )

  it.each(Object.entries(BUFFER_WRITES))(
    'are taken through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it('are passed to the window the plugin runs in, and to none where the window made none', () => {
    const { context, buffers } = createContext(['vault:write'])
    context.vault.buffers!.track({} as never)

    expect(buffers.track).toHaveBeenCalledOnce()
    expect(
      createPluginContext('md.numen.sample', {} as unknown as PluginContextHost, {
        permissions: ['vault:write'],
      }),
    ).not.toHaveProperty('buffers')
  })
})

describe('the registrations of a plugin in the open files', () => {
  it('are released when the plugin is disposed', () => {
    const { context, buffers } = createContext(['vault:write'])
    const released = vi.fn()
    vi.mocked(buffers.track).mockReturnValue({ dispose: released })
    vi.mocked(buffers.addFlushHandler).mockReturnValue({ dispose: released })

    context.vault.buffers!.track({} as never)
    context.vault.buffers!.addFlushHandler(vi.fn(() => Promise.resolve()))
    expect(context.plugin.subscriptions).toHaveLength(2)
    for (const one of [...context.plugin.subscriptions]) one.dispose()

    expect(released).toHaveBeenCalledTimes(2)
  })
})

describe('a file a plugin lists and writes', () => {
  it.each(Object.entries(FILE_WRITES))(
    'fails through %s without vault:write',
    async (name, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "vault:write" for files.${name}`,
      )
    },
  )

  it.each(Object.entries(FILE_WRITES))(
    'is written through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it('is listed with vault:read and without vault:write', async () => {
    const { context, files } = createContext(['vault:read'])

    await context.vault.files!.list('Notes')

    expect(files.list).toHaveBeenCalledWith('Notes')
  })

  it.each([
    ['vault:write', ['vault:write']],
    ['no permission', []],
  ])('fails to be listed with %s', async (_, permissions) => {
    const { context, files } = createContext(permissions)

    await expect(context.vault.files!.list('Notes')).rejects.toThrowError(
      'Plugin "md.numen.sample" lacks required permission "vault:read" for files.list',
    )
    expect(files.list).not.toHaveBeenCalled()
  })
})

describe('a command a plugin runs over the vault', () => {
  const target = { path: 'a.md', title: 'A', isFolder: false, others: [], tab: '' }
  const WRITES = [
    'app:note.trash',
    'app:note.createChild',
    'app:note.rename',
    'md.numen.engines.transcription:transcript.create',
  ]

  it.each(WRITES)('runs %s through the commands of the window with no permission', async (id) => {
    const { context, commands } = createContext([])

    await context.ui.commands!.run(id, target)

    expect(commands.getRuns()).toStrictEqual([{ command: id, target, itemId: undefined }])
  })

  it('reads the commands of the window and their items with no permission', () => {
    const { context, commands } = createContext([])
    const handler = vi.fn()
    context.ui.commands!.watch(handler)

    commands.setIds(['app:vault.create'])

    expect(context.ui.commands!.getIds()).toStrictEqual(['app:vault.create'])
    expect(handler).toHaveBeenLastCalledWith(['app:vault.create'])
  })

  it('opens a file and shows an error without vault:write', () => {
    const { context, workspace } = createContext([])

    context.ui.workspace!.openFile('a.md')
    context.ui.workspace!.showError('no')

    expect(workspace.files.open).toHaveBeenCalledWith('a.md')
    expect(workspace.messages.showError).toHaveBeenCalledWith('no')
  })
})

const getAgent = (context: PluginContext): AgentHost => {
  if (!context.work.agent) throw new Error('the context was handed no agent')
  return context.work.agent
}

const ASKING: Record<string, [string, Call]> = {
  ask: [
    'vault:read',
    (c) =>
      getAgent(c).ask({
        question: 'q',
        focus: '',
        conversation: 'c',
        signal: new AbortController().signal,
      }),
  ],
  finish: ['vault:read', (c) => getAgent(c).finish('c')],
  resolveLinks: ['vault:read', (c) => getAgent(c).resolveLinks('[[A]]')],
}

describe('the agent a plugin asks', () => {
  it.each(Object.entries(ASKING))(
    'fails through %s without its permission',
    async (name, [permission, call]) => {
      const { context } = createContext([])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        `Plugin "md.numen.sample" lacks required permission "${permission}" for agent.${name}`,
      )
    },
  )

  it.each(Object.entries(ASKING))(
    'is reached through %s with its permission',
    async (name, [permission, call]) => {
      const { context, agent } = createContext([permission])

      await call(context)

      expect(agent[name as 'ask' | 'finish' | 'resolveLinks']).toHaveBeenCalled()
    },
  )

  it('reads why it cannot be reached without a permission', () => {
    const { context } = createContext([])

    expect(getAgent(context).getUnreachable()).toBe('')
  })

  it('reads, writes and lists the settings under agent, and no others', async () => {
    const { context, agent } = createContext([])
    const request = getAgent(context)

    await request.writeSetting(['app', 'agent', 'use'], 'zeta')
    request.getSetting(['app', 'agent', 'use'])
    request.getModelsAt(['app', 'agent', 'zeta', 'model'])

    expect(agent.writeSetting).toHaveBeenCalledWith(['app', 'agent', 'use'], 'zeta')
    expect(agent.getSetting).toHaveBeenCalledWith(['app', 'agent', 'use'])
    expect(agent.getModelsAt).toHaveBeenCalledWith(['app', 'agent', 'zeta', 'model'])

    const elsewhere = ['md.numen.other', 'enabled']
    expect(request.getSetting(elsewhere)).toBeUndefined()
    expect(request.getModelsAt(elsewhere)).toStrictEqual([])
    await expect(request.writeSetting(elsewhere, false)).rejects.toThrowError(
      'agent.writeSetting is limited to the agent settings, not "md.numen.other.enabled"',
    )
    expect(agent.getSetting).toHaveBeenCalledTimes(1)
    expect(agent.writeSetting).toHaveBeenCalledTimes(1)
  })
})
