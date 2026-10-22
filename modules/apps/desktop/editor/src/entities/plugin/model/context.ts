/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  AgentHost,
  AgentProviderRegistration,
  BufferHost,
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginManifest,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
  VaultFiles,
  VaultNotes,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { PluginContextHost } from './types'

function trackSubscription(subscriptions: Disposable[], disposable: Disposable): Disposable {
  subscriptions.push(disposable)
  return {
    dispose() {
      const index = subscriptions.indexOf(disposable)
      if (index !== -1) {
        subscriptions.splice(index, 1)
      }
      disposable.dispose()
    },
  }
}

interface PermissionTarget {
  readonly pluginId: string
  readonly manifest?: PluginManifest | { permissions?: readonly string[] } | undefined
}

function checkPermission(target: PermissionTarget, permission: string, operation: string): void {
  if (!(target.manifest?.permissions?.includes(permission) ?? false)) {
    throw new Error(
      `Plugin "${target.pluginId}" lacks required permission "${permission}" for ${operation}`,
    )
  }
}

/** The commands of the window over a file that change nothing in the vault. */
const READ_ONLY_COMMANDS: ReadonlySet<string> = new Set(['read', 'travel', 'copy', 'ask'])

function guardNotes(notes: VaultNotes, requireWrite: (operation: string) => void): VaultNotes {
  return {
    ...notes,
    createUntitledNote: async () => {
      requireWrite('notes.createUntitledNote')
      return notes.createUntitledNote()
    },
    createInRelation: async (from, relation) => {
      requireWrite('notes.createInRelation')
      return notes.createInRelation(from, relation)
    },
    createWithTitle: async (title, from, relation) => {
      requireWrite('notes.createWithTitle')
      return notes.createWithTitle(title, from, relation)
    },
    join: async (...args: Parameters<VaultNotes['join']>) => {
      requireWrite('notes.join')
      return notes.join(...args)
    },
    replaceLinks: async (first, second, links) => {
      requireWrite('notes.replaceLinks')
      return notes.replaceLinks(first, second, links)
    },
  }
}

function guardFiles(files: VaultFiles, requireWrite: (operation: string) => void): VaultFiles {
  return {
    ...files,
    move: async (from, to) => {
      requireWrite('files.move')
      return files.move(from, to)
    },
    createFolder: async (path) => {
      requireWrite('files.createFolder')
      return files.createFolder(path)
    },
    createNote: async (folder) => {
      requireWrite('files.createNote')
      return files.createNote(folder)
    },
    createDeck: async (folder, name) => {
      requireWrite('files.createDeck')
      return files.createDeck(folder, name)
    },
    createStencil: async (folder, name) => {
      requireWrite('files.createStencil')
      return files.createStencil(folder, name)
    },
    createPreset: async (folder, name) => {
      requireWrite('files.createPreset')
      return files.createPreset(folder, name)
    },
    importUrl: async (folder, url) => {
      requireWrite('files.importUrl')
      return files.importUrl(folder, url)
    },
    importUrls: async (folder, urls) => {
      requireWrite('files.importUrls')
      return files.importUrls(folder, urls)
    },
  }
}

/** Whether a setting path lies in the agent subtree of the settings. */
const isAgentPath = (path: readonly string[]): boolean => path[0] === 'agent'

/**
 * The agent of the window: asking it and naming vault files need vault:read, and
 * the settings it reads and writes are the ones under `agent`.
 */
function guardAgent(
  agent: AgentHost,
  requirePerm: (permission: string, operation: string) => void,
): AgentHost {
  return {
    ...agent,
    getSetting: (path) => (isAgentPath(path) ? agent.getSetting(path) : undefined),
    writeSetting: async (path, value) => {
      if (!isAgentPath(path)) {
        throw new Error(
          `agent.writeSetting is limited to the agent settings, not "${path.join('.')}"`,
        )
      }
      await agent.writeSetting(path, value)
    },
    getModelsAt: (path) => (isAgentPath(path) ? agent.getModelsAt(path) : []),
    ask: (query) => {
      requirePerm('vault:read', 'agent.ask')
      return agent.ask(query)
    },
    finish: (conversation) => {
      requirePerm('vault:read', 'agent.finish')
      return agent.finish(conversation)
    },
    resolve: (content) => {
      requirePerm('vault:read', 'agent.resolve')
      return agent.resolve(content)
    },
  }
}

function guardBuffers(
  buffers: BufferHost,
  requireWrite: (operation: string) => void,
  add: (disposable: Disposable) => Disposable,
): BufferHost {
  return {
    track: (open) => {
      requireWrite('buffers.track')
      return add(buffers.track(open))
    },
    addFlushHandler: (handler) => {
      requireWrite('buffers.addFlushHandler')
      return add(buffers.addFlushHandler(handler))
    },
  }
}

function guardWorkspace(
  workspace: WorkspaceHost,
  requireWrite: (operation: string) => void,
): WorkspaceHost {
  return {
    ...workspace,
    runCommand: (id, target) => {
      if (!READ_ONLY_COMMANDS.has(id)) requireWrite(`workspace.runCommand ${id}`)
      workspace.runCommand(id, target)
    },
    runPathCommand: (id, target) => {
      if (!READ_ONLY_COMMANDS.has(id)) requireWrite(`workspace.runPathCommand ${id}`)
      workspace.runPathCommand(id, target)
    },
  }
}

type Services = Pick<PluginContext, 'workspace' | 'notes' | 'files' | 'agent' | 'buffers'>

/** A service the window made, guarded; nothing where the window made none. */
function guardService<K extends keyof Services, S>(
  key: K,
  service: S | undefined,
  guard: (service: S) => NonNullable<Services[K]>,
): Pick<Services, K> | Record<never, never> {
  if (service === undefined) return {}
  const out: Partial<Record<K, NonNullable<Services[K]>>> = { [key]: guard(service) }
  return out as Pick<Services, K>
}

function guardServices(
  host: PluginContextHost,
  requirePerm: (permission: string, operation: string) => void,
  add: (disposable: Disposable) => Disposable,
): Services {
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    ...guardService('workspace', host.getWorkspace?.(), (one) => guardWorkspace(one, requireWrite)),
    ...guardService('notes', host.getNotes?.(), (one) => guardNotes(one, requireWrite)),
    ...guardService('files', host.getFiles?.(), (one) => guardFiles(one, requireWrite)),
    ...guardService('agent', host.getAgent?.(), (one) => guardAgent(one, requirePerm)),
    ...guardService('buffers', host.getBuffers?.(), (one) => guardBuffers(one, requireWrite, add)),
  }
}

export function createPluginContext(
  pluginId: string,
  host: PluginContextHost,
  manifest?: PluginManifest | { permissions?: readonly string[] },
): PluginContext {
  const subscriptions: Disposable[] = []
  const add = (d: Disposable) => trackSubscription(subscriptions, d)
  const target: PermissionTarget = { pluginId, manifest }
  const requirePerm = (p: string, op: string) => checkPermission(target, p, op)

  return {
    pluginId,
    subscriptions,
    ...guardServices(host, requirePerm, add),
    hasPermission: (p: string) => manifest?.permissions?.includes(p) ?? false,
    registerReader: (reg: ReaderRegistration) => add(host.registerReader(pluginId, reg)),
    registerTabKind: <State>(reg: TabKindRegistration<State>) =>
      add(host.registerTabKind(pluginId, reg)),
    registerCommand: (reg: CommandRegistration) => add(host.registerCommand(pluginId, reg)),
    registerAgentProvider: (reg: AgentProviderRegistration) =>
      add(host.registerAgentProvider(pluginId, reg)),
    getSetting: <T = unknown>(key: string) => host.getSetting<T>(pluginId, key),
    onSettingChange: <T = unknown>(key: string, h: (v: T) => void) =>
      add(host.onSettingChange<T>(pluginId, key, h)),
    onVaultChange: (h: (e: VaultChangeEvent) => void | Promise<void>) => {
      requirePerm('vault:read', 'onVaultChange')
      return add(host.onVaultChange(pluginId, h))
    },
    onFileSelect: (h: (e: FileSelectEvent) => void | Promise<void>) => {
      requirePerm('vault:read', 'onFileSelect')
      return add(host.onFileSelect(pluginId, h))
    },
    emitVaultChange: (e: VaultChangeEvent) => {
      requirePerm('vault:write', 'emitVaultChange')
      host.emitVaultChange(pluginId, e)
    },
    emitFileSelect: (e: FileSelectEvent) => {
      requirePerm('vault:read', 'emitFileSelect')
      host.emitFileSelect(pluginId, e)
    },
    openTab: (kind: string, targetPath?: string) => host.openTab?.(kind, targetPath),
    closeTab: (id: string) => {
      host.closeTab?.(id)
    },
  }
}
