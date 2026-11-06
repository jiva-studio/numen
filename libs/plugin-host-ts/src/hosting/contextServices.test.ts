/**
 * What a plugin may do through the window, the notes and the files it is
 * handed: the reads are open, the writes need `vault:write`.
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
import { createPluginContext } from './context'
import type { PluginContextHost } from './types'

const createServices = () => {
  const workspace = {
    runCommand: vi.fn(),
    openFile: vi.fn(),
    showError: vi.fn(),
  } as unknown as WorkspaceHost
  const notes = {
    getHeadings: vi.fn(() => Promise.resolve(new Map())),
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
  const host = {
    getAgent: () => agent,
    getBuffers: () => buffers,
    getWorkspace: () => workspace,
    getNotes: () => notes,
    getRelations: () => relations,
    getFiles: () => files,
  } as unknown as PluginContextHost
  return { host, workspace, notes, relations, files, agent, buffers }
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

    expect(context.relations!.roles).toStrictEqual(['parent', 'child', 'jump'])
  })
})

const FILE_WRITES: Record<string, Call> = {
  move: (c) => c.files!.move('a.md', 'b.md'),
  createFolder: (c) => c.files!.createFolder('dir'),
  createNote: (c) => c.files!.createNote('dir'),
}

describe('a note a plugin writes', () => {
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

  it('is read without vault:write', async () => {
    const { context, notes } = createContext(['vault:read'])

    await context.notes!.getHeadings(['a.md'])

    expect(notes.getHeadings).toHaveBeenCalledWith(['a.md'])
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
    context.buffers!.track({} as never)

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

    context.buffers!.track({} as never)
    context.buffers!.addFlushHandler(vi.fn(() => Promise.resolve()))
    expect(context.subscriptions).toHaveLength(2)
    for (const one of [...context.subscriptions]) one.dispose()

    expect(released).toHaveBeenCalledTimes(2)
  })
})

describe('a file a plugin writes', () => {
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

  it('is listed without vault:write', async () => {
    const { context, files } = createContext([])

    await context.files!.list('dir')

    expect(files.list).toHaveBeenCalledWith('dir')
  })
})

describe('a command a plugin runs over the vault', () => {
  const target = { path: 'a.md', title: 'A' }

  it.each([
    'app:note.trash',
    'app:note.createChild',
    'app:note.rename',
    'md.numen.engines.transcription:transcript.create',
  ])('fails for %s without vault:write', (id) => {
    const { context, workspace } = createContext(['vault:read'])

    expect(() => context.workspace!.runCommand(id, target)).toThrowError(
      `Plugin "md.numen.sample" lacks required permission "vault:write" for workspace.runCommand ${id}`,
    )
    expect(workspace.runCommand).not.toHaveBeenCalled()
  })

  it.each([
    'app:note.trash',
    'app:note.createChild',
    'app:note.rename',
    'md.numen.engines.transcription:transcript.create',
  ])('runs %s with vault:write', (id) => {
    const { context, workspace } = createContext(['vault:write'])

    context.workspace!.runCommand(id, target)

    expect(workspace.runCommand).toHaveBeenCalledWith(id, target)
  })

  it.each([
    'app:note.open',
    'md.numen.views.plex.graph:graph.showNote',
    'app:note.copyPath',
    'app:settings.open',
    'md.numen.views.agent.chat:note.ask',
  ])('runs %s without vault:write', (id) => {
    const { context, workspace } = createContext([])

    context.workspace!.runCommand(id, target)

    expect(workspace.runCommand).toHaveBeenCalledWith(id, target)
  })

  it('opens a file and shows an error without vault:write', () => {
    const { context, workspace } = createContext([])

    context.workspace!.openFile('a.md')
    context.workspace!.showError('no')

    expect(workspace.openFile).toHaveBeenCalledWith('a.md')
    expect(workspace.showError).toHaveBeenCalledWith('no')
  })
})

/** The agent a context was handed, which every test of it was given. */
const getAgent = (context: PluginContext): AgentHost => {
  if (!context.agent) throw new Error('the context was handed no agent')
  return context.agent
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
