/**
 * What a plugin may do through the window, the notes and the files it is
 * handed: the reads are open, the writes need `vault:write`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it, vi } from 'vitest'
import type {
  AgentHost,
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
  } as unknown as VaultNotes
  const files = {
    list: vi.fn(() => Promise.resolve([])),
    move: vi.fn(() => Promise.resolve()),
    createFolder: vi.fn(() => Promise.resolve()),
    createNote: vi.fn(() => Promise.resolve('a.md')),
    createDeck: vi.fn(() => Promise.resolve('a.deck')),
    createStencil: vi.fn(() => Promise.resolve('a.stencil')),
    createPreset: vi.fn(() => Promise.resolve('a.preset')),
    importUrl: vi.fn(() => Promise.resolve('u.md')),
    importUrls: vi.fn(() => Promise.resolve(['u.md'])),
  } as unknown as VaultFiles
  const agent = {
    ask: vi.fn(),
    finish: vi.fn(() => Promise.resolve()),
    resolve: vi.fn(() => Promise.resolve(new Map())),
    getUnreachable: vi.fn(() => ''),
  } as unknown as AgentHost
  const host = {
    getAgent: () => agent,
    getWorkspace: () => workspace,
    getNotes: () => notes,
    getFiles: () => files,
  } as unknown as PluginContextHost
  return { host, workspace, notes, files, agent }
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
}

const FILE_WRITES: Record<string, Call> = {
  move: (c) => c.files!.move('a.md', 'b.md'),
  createFolder: (c) => c.files!.createFolder('dir'),
  createNote: (c) => c.files!.createNote('dir'),
  createDeck: (c) => c.files!.createDeck('dir', 'D'),
  createStencil: (c) => c.files!.createStencil('dir', 'S'),
  createPreset: (c) => c.files!.createPreset('dir', 'P'),
  importUrl: (c) => c.files!.importUrl('dir', 'https://example.test'),
  importUrls: (c) => c.files!.importUrls('dir', ['https://example.test']),
}

describe('a note a plugin writes', () => {
  it.each(Object.entries(NOTE_WRITES))(
    'is refused through %s without vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(/vault:write/)
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

describe('a file a plugin writes', () => {
  it.each(Object.entries(FILE_WRITES))(
    'is refused through %s without vault:write',
    async (_, call) => {
      const { context } = createContext(['vault:read'])

      await expect(Promise.resolve().then(() => call(context))).rejects.toThrowError(/vault:write/)
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
  const row = { paths: ['a.md'], name: 'a', source: 'note' as const }

  it.each(['remove', 'child', 'title', 'transcribe'])(
    'is refused for %s without vault:write',
    (id) => {
      const { context, workspace } = createContext(['vault:read'])

      expect(() => context.workspace!.runCommand(id, target)).toThrowError(/vault:write/)
      expect(() => context.workspace!.runPathCommand(id, row)).toThrowError(/vault:write/)
      expect(workspace.runCommand).not.toHaveBeenCalled()
      expect(workspace.runPathCommand).not.toHaveBeenCalled()
    },
  )

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

const ASKING: Record<string, [string, Call]> = {
  ask: [
    'ai:complete',
    (c) =>
      c.agent!.ask({
        question: 'q',
        focus: '',
        conversation: 'c',
        signal: new AbortController().signal,
      }),
  ],
  finish: ['ai:complete', (c) => c.agent!.finish('c')],
  resolve: ['vault:read', (c) => c.agent!.resolve(['name://A'])],
}

describe('the agent a plugin asks', () => {
  it.each(Object.entries(ASKING))(
    'is refused through %s without its permission',
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

      expect(agent[name as 'ask' | 'finish' | 'resolve']).toHaveBeenCalled()
    },
  )

  it('reads why it cannot be reached without a permission', () => {
    const { context } = createContext([])

    expect(context.agent!.getUnreachable()).toBe('')
  })
})

const PLUGINS = resolve(import.meta.dirname, '../../../../../../../plugins')

const readPermissions = (plugin: string): readonly string[] =>
  JSON.parse(readFileSync(resolve(PLUGINS, plugin, 'manifest.json'), 'utf8')).permissions

describe('the manifest of a plugin that writes through the window', () => {
  it('view-file-tree declares what the files it moves and makes need', async () => {
    const { context } = createContext(readPermissions('view-file-tree'))

    await Promise.all(Object.values(FILE_WRITES).map((call) => call(context)))
    context.workspace!.runPathCommand('remove', {
      paths: ['a.md'],
      name: 'a',
      source: 'note',
    })
  })

  it('view-agent-chat declares what the agent it asks needs', async () => {
    const { context } = createContext(readPermissions('view-agent-chat'))

    await Promise.all(Object.values(ASKING).map(([, call]) => call(context)))
  })

  it('view-plex-graph declares what the notes it joins and makes need', async () => {
    const { context } = createContext(readPermissions('view-plex-graph'))

    await Promise.all(Object.values(NOTE_WRITES).map((call) => call(context)))
    context.workspace!.runCommand('child', { path: 'a.md', title: 'A' })
  })
})
