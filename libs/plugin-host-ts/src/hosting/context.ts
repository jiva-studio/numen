/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  AgentHost,
  AgentProviderRegistration,
  BufferHost,
  CommandRegistration,
  Disposable,
  DropRegistration,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultTexts,
  ViewRegistration,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createExternalOpener } from '../externalOpener'
import { createSlotContext } from './slotContext'
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
  readonly manifest?: Manifest | { permissions?: readonly string[] } | undefined
}

function checkPermission(target: PermissionTarget, permission: string, operation: string): void {
  if (!(target.manifest?.permissions?.includes(permission) ?? false)) {
    throw new Error(
      `Plugin "${target.pluginId}" lacks required permission "${permission}" for ${operation}`,
    )
  }
}

function guardNotes(
  notes: VaultNotes,
  requirePerm: (permission: string, operation: string) => void,
): VaultNotes {
  const requireRead = (operation: string) => requirePerm('vault:read', operation)
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    getNeighbourhood: async (path) => {
      requireRead('notes.getNeighbourhood')
      return notes.getNeighbourhood(path)
    },
    getHeadings: async (paths) => {
      requireRead('notes.getHeadings')
      return notes.getHeadings(paths)
    },
    findNotes: async (query, limit) => {
      requireRead('notes.findNotes')
      return notes.findNotes(query, limit)
    },
    createUntitledNote: async () => {
      requireWrite('notes.createUntitledNote')
      return notes.createUntitledNote()
    },
    read: async (path) => {
      requireRead('notes.read')
      return notes.read(path)
    },
    write: async (path, body) => {
      requireWrite('notes.write')
      return notes.write(path, body)
    },
    readProperties: async (path) => {
      requireRead('notes.readProperties')
      return notes.readProperties(path)
    },
  }
}

function guardRelations(
  relations: VaultRelations,
  requirePerm: (permission: string, operation: string) => void,
): VaultRelations {
  const requireRead = (operation: string) => requirePerm('vault:read', operation)
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    roles: relations.roles,
    getConnection: async (first, second, type) => {
      requireRead('relations.getConnection')
      return relations.getConnection(first, second, type)
    },
    saveConnection: async (connection) => {
      requireWrite('relations.saveConnection')
      return relations.saveConnection(connection)
    },
    removeSide: async (from, to, type) => {
      requireWrite('relations.removeSide')
      return relations.removeSide(from, to, type)
    },
    removeConnection: async (first, second, type) => {
      requireWrite('relations.removeConnection')
      return relations.removeConnection(first, second, type)
    },
    join: async (from, to, side) => {
      requireWrite('relations.join')
      return relations.join(from, to, side)
    },
    createRelated: async (request) => {
      requireWrite('relations.createRelated')
      return relations.createRelated(request)
    },
    listBacklinks: async (path) => {
      requireRead('relations.listBacklinks')
      return relations.listBacklinks(path)
    },
  }
}

function guardFiles(
  files: VaultFiles,
  requirePerm: (permission: string, operation: string) => void,
): VaultFiles {
  const requireRead = (operation: string) => requirePerm('vault:read', operation)
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    list: async (folder) => {
      requireRead('files.list')
      return files.list(folder)
    },
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
  }
}

function guardTexts(
  texts: VaultTexts,
  requirePerm: (permission: string, operation: string) => void,
): VaultTexts {
  return {
    readLiveText: (path) => {
      requirePerm('vault:read', 'texts.readLiveText')
      return texts.readLiveText(path)
    },
    createEdit: (path, sourceVersionId, segments) => {
      requirePerm('vault:write', 'texts.createEdit')
      return texts.createEdit(path, sourceVersionId, segments)
    },
  }
}

const isAgentPath = (path: readonly string[]): boolean => path[0] === 'app' && path[1] === 'agent'

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
    resolveLinks: (text, vault) => {
      requirePerm('vault:read', 'agent.resolveLinks')
      return agent.resolveLinks(text, vault)
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

type Services = Pick<
  PluginContext,
  'workspace' | 'notes' | 'relations' | 'files' | 'agent' | 'buffers'
>

function guardService<K extends keyof Services, S>(
  key: K,
  service: S | undefined,
  guard: (service: S) => NonNullable<Services[K]>,
): Pick<Services, K> | Record<never, never> {
  if (service === undefined) return {}
  const out: Partial<Record<K, NonNullable<Services[K]>>> = {}
  out[key] = guard(service)
  return out
}

function guardServices(
  host: PluginContextHost,
  requirePerm: (permission: string, operation: string) => void,
  add: (disposable: Disposable) => Disposable,
): Services {
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    ...guardService('workspace', host.getWorkspace?.(), (one: WorkspaceHost) => one),
    ...guardService('notes', host.getNotes?.(), (one) => guardNotes(one, requirePerm)),
    ...guardService('relations', host.getRelations?.(), (one) => guardRelations(one, requirePerm)),
    ...guardService('files', host.getFiles?.(), (one) => guardFiles(one, requirePerm)),
    ...guardService('agent', host.getAgent?.(), (one) => guardAgent(one, requirePerm)),
    ...guardService('buffers', host.getBuffers?.(), (one) => guardBuffers(one, requireWrite, add)),
  }
}

/** The target the editor activates its plugins in. */
const EDITOR_TARGET = 'desktop.editor'

/** A view context that names a note to read, such as the one of the note editor. */
function hasNotePath(context: unknown): boolean {
  return typeof context === 'object' && context !== null && 'path' in context && !!context.path
}

type Track = (disposable: Disposable) => Disposable
type RequirePermission = (permission: string, operation: string) => void

function createRegistrations(
  pluginId: string,
  host: PluginContextHost,
  add: Track,
  requirePerm: RequirePermission,
) {
  return {
    registerReader: (reg: ReaderRegistration) => add(host.registerReader(pluginId, reg)),
    registerSettingsView: (reg: SettingsViewRegistration) =>
      add(host.registerSettingsView(pluginId, reg)),
    registerTabKind: <State>(reg: TabKindRegistration<State>) =>
      add(host.registerTabKind(pluginId, reg)),
    registerCommand: (reg: CommandRegistration) => add(host.registerCommand(pluginId, reg)),
    registerAgentProvider: (reg: AgentProviderRegistration) =>
      add(host.registerAgentProvider(pluginId, reg)),
    registerView: (reg: ViewRegistration) => add(host.registerView(pluginId, reg)),
    mountView: (capability: string, element: HTMLElement, context: unknown) => {
      if (hasNotePath(context)) requirePerm('vault:read', 'mountView')
      const view = host.mountView(capability, element, context)
      return { focus: () => view.focus(), ...add(view) }
    },
    registerDropHandler: (reg: DropRegistration) => add(host.registerDropHandler(pluginId, reg)),
  }
}

/** What the events of a plugin are tracked and guarded with. */
interface EventGuard {
  readonly requirePerm: RequirePermission
  readonly add: Track
}

function createEvents(pluginId: string, host: PluginContextHost, guard: EventGuard) {
  const { requirePerm, add } = guard
  return {
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
  }
}

function createWindowServices(
  pluginId: string,
  host: PluginContextHost,
  requirePerm: RequirePermission,
) {
  const notices = host.getNotices?.(pluginId)
  const tasks = host.getTasks?.(pluginId)
  const backgroundTasks = host.getBackgroundTasks?.()
  const texts = host.getTexts?.(pluginId)
  return {
    ...(notices ? { notices } : {}),
    ...(tasks ? { tasks } : {}),
    ...(backgroundTasks ? { backgroundTasks } : {}),
    ...(texts ? { texts: guardTexts(texts, requirePerm) } : {}),
    ...createFrameServices(host),
  }
}

export function createPluginContext(
  pluginId: string,
  host: PluginContextHost,
  manifest?: Manifest | { permissions?: readonly string[] },
): PluginContext {
  const subscriptions: Disposable[] = []
  const add = (d: Disposable) => trackSubscription(subscriptions, d)
  const target: PermissionTarget = { pluginId, manifest }
  const requirePerm = (p: string, op: string) => checkPermission(target, p, op)

  return {
    pluginId,
    target: EDITOR_TARGET,
    app: host.app,
    openExternal: createExternalOpener(host.openExternal),
    subscriptions,
    ...guardServices(host, requirePerm, add),
    ...createWindowServices(pluginId, host, requirePerm),
    hasPermission: (p: string) => manifest?.permissions?.includes(p) ?? false,
    ...createRegistrations(pluginId, host, add, requirePerm),
    ...createSlotContext(
      { pluginId, manifest: manifest && 'contributes' in manifest ? manifest : undefined },
      host,
      add,
    ),
    drops: host.drops,
    ...createEvents(pluginId, host, { requirePerm, add }),
    openTab: (kind: string, targetPath?: string) => host.openTab?.(kind, targetPath),
    navigateTab: (kind: string, targetPath: string) => host.navigateTab?.(kind, targetPath),
    closeTab: (id: string) => {
      host.closeTab?.(id)
    },
  }
}

function createFrameServices(host: PluginContextHost) {
  const commands = host.getCommands?.()
  const vaults = host.getVaults?.()
  const window = host.getWindow?.()
  return {
    ...(commands ? { commands } : {}),
    ...(vaults ? { vaults } : {}),
    ...(window ? { window } : {}),
  }
}
