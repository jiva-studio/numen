/**
 * What a plugin may do through the window, the notes and the files it is
 * handed: the reads are open, the writes need `vault:write`.
 */
import { describe, expect, it, vi } from 'vitest'
import type {
  AgentHost,
  BufferHost,
  PluginContext,
  VaultFiles,
  VaultNotes,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import { createPluginContext } from './context'
import type { PluginContextHost } from './types'

const createServices = () => {
  const workspace = {
    runCommand: vi.fn(),
    runPathCommand: vi.fn(),
    openFile: vi.fn(),
    showError: vi.fn(),
  } as unknown as WorkspaceHost
  const notes = {
    getHeadings: vi.fn(() => Promise.resolve(new Map())),
    createUntitledNote: vi.fn(() => Promise.resolve('Untitled.md')),
    createInRelation: vi.fn(() => Promise.resolve(null)),
    createWithTitle: vi.fn(() => Promise.resolve(null)),
    join: vi.fn(() => Promise.resolve(true)),
    replaceLinks: vi.fn(() => Promise.resolve(true)),
    write: vi.fn(() => Promise.resolve()),
  } as unknown as VaultNotes
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
    getFiles: () => files,
  } as unknown as PluginContextHost
  return { host, workspace, notes, files, agent, buffers }
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
  createInRelation: (c) => c.notes!.createInRelation('a.md', 'child'),
  createWithTitle: (c) => c.notes!.createWithTitle('T', 'a.md', 'child'),
  join: (c) => c.notes!.join('a.md', 'b.md', 'child'),
  replaceLinks: (c) => c.notes!.replaceLinks('a.md', 'b.md', []),
  write: (c) => c.notes!.write('a.md', 'Prose.'),
}

const FILE_WRITES: Record<string, Call> = {
  move: (c) => c.files!.move('a.md', 'b.md'),
  createFolder: (c) => c.files!.createFolder('dir'),
  createNote: (c) => c.files!.createNote('dir'),
}

describe('a note a plugin writes', () => {
  it.each(Object.entries(NOTE_WRITES))('fails through %s without vault:write', async (_, call) => {
    const { context } = createContext(['vault:read'])

    await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(/vault:write/)
  })

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
  it.each(Object.entries(BUFFER_WRITES))('fail through %s without vault:write', async (_, call) => {
    const { context } = createContext(['vault:read'])

    await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(/vault:write/)
  })

  it.each(Object.entries(BUFFER_WRITES))(
    'are taken through %s with vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:write'])

      await expect(Promise.resolve().then(() => call(context))).resolves.not.toThrow()
    },
  )

  it('are handed over to the window the plugin runs in, and to none where the window made none', () => {
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
  it.each(Object.entries(FILE_WRITES))('fails through %s without vault:write', async (_, call) => {
    const { context } = createContext(['vault:read'])

    await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(/vault:write/)
  })

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
  const row = { paths: ['a.md'], name: 'a', isNote: true, isFolder: false }

  it.each(['remove', 'child', 'title', 'transcribe'])('fails for %s without vault:write', (id) => {
    const { context, workspace } = createContext(['vault:read'])

    expect(() => context.workspace!.runCommand(id, target)).toThrowError(/vault:write/)
    expect(() => context.workspace!.runPathCommand(id, row)).toThrowError(/vault:write/)
    expect(workspace.runCommand).not.toHaveBeenCalled()
    expect(workspace.runPathCommand).not.toHaveBeenCalled()
  })

  it.each(['remove', 'child', 'title', 'transcribe'])('runs %s with vault:write', (id) => {
    const { context, workspace } = createContext(['vault:write'])

    context.workspace!.runCommand(id, target)
    context.workspace!.runPathCommand(id, row)

    expect(workspace.runCommand).toHaveBeenCalledWith(id, target)
    expect(workspace.runPathCommand).toHaveBeenCalledWith(id, row)
  })

  it.each(['read', 'travel', 'copy', 'ask'])('runs %s without vault:write', (id) => {
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
    async (_, [permission, call]) => {
      const { context } = createContext([])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(
        new RegExp(permission),
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

    await request.writeSetting(['agent', 'use'], 'zeta')
    request.getSetting(['agent', 'use'])
    request.getModelsAt(['agent', 'zeta', 'model'])

    expect(agent.writeSetting).toHaveBeenCalledWith(['agent', 'use'], 'zeta')
    expect(agent.getSetting).toHaveBeenCalledWith(['agent', 'use'])
    expect(agent.getModelsAt).toHaveBeenCalledWith(['agent', 'zeta', 'model'])

    const elsewhere = ['plugins', 'md.numen.other', 'enabled']
    expect(request.getSetting(elsewhere)).toBeUndefined()
    expect(request.getModelsAt(elsewhere)).toStrictEqual([])
    await expect(request.writeSetting(elsewhere, false)).rejects.toThrowError(/agent settings/)
    expect(agent.getSetting).toHaveBeenCalledTimes(1)
    expect(agent.writeSetting).toHaveBeenCalledTimes(1)
  })
})
