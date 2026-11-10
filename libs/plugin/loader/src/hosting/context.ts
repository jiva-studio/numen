/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  BufferHost,
  Disposable,
  FileSelectEvent,
  PluginContext,
  PluginEvents,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
  VaultChangeEvent,
  VaultFiles,
  VaultNotes,
  VaultRelations,
  VaultTexts,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createExternalOpener } from '../externalOpener'
import { guardAgent } from './guards/agentGuard'
import { checkPermission, type PermissionTarget } from './guards/permission'
import {
  createMountView,
  createRegistrations,
  type RequirePermission,
  type Track,
} from './guards/registrations'
import { createSlotViewRegistration } from './slotContext'
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

function guardNotes(
  notes: VaultNotes,
  requirePerm: (permission: string, operation: string) => void,
): VaultNotes {
  const requireRead = (operation: string) => requirePerm('vault:read', operation)
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return {
    getNeighborhood: async (path) => {
      requireRead('notes.getNeighborhood')
      return notes.getNeighborhood(path)
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

/** The target the editor activates its plugins in. */
const EDITOR_TARGET = 'desktop.editor'

/** What the events of a plugin are tracked and guarded with. */
interface EventGuard {
  readonly requirePerm: RequirePermission
  readonly add: Track
}

function createSettings(pluginId: string, host: PluginContextHost, add: Track): PluginSettings {
  return {
    get: <T = unknown>(key: string) => host.getSetting<T>(pluginId, key),
    onChange: <T = unknown>(key: string, h: (v: T) => void) =>
      add(host.onSettingChange<T>(pluginId, key, h)),
  }
}

function createEvents(pluginId: string, host: PluginContextHost, guard: EventGuard): PluginEvents {
  const { requirePerm, add } = guard
  return {
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

function createTabs(host: PluginContextHost): PluginTabs {
  return {
    open: (kind: string, targetPath?: string) => host.openTab?.(kind, targetPath),
    navigate: (kind: string, targetPath: string) => host.navigateTab?.(kind, targetPath),
    close: (id: string) => {
      host.closeTab?.(id)
    },
  }
}

function guarded<S, G>(service: S | undefined, guard: (service: S) => G): G | undefined {
  return service === undefined ? undefined : guard(service)
}

function pickDefined<T extends object>(parts: { [K in keyof T]: T[K] | undefined }): T {
  const picked: Partial<T> = {}
  for (const key of Object.keys(parts) as (keyof T)[]) {
    const value = parts[key]
    if (value !== undefined) picked[key] = value
  }
  return picked as T
}

function createVault(
  pluginId: string,
  host: PluginContextHost,
  guard: { readonly requirePerm: RequirePermission; readonly add: Track },
): PluginVault {
  const { requirePerm, add } = guard
  const requireWrite = (operation: string) => requirePerm('vault:write', operation)
  return pickDefined<PluginVault>({
    notes: guarded(host.getNotes?.(), (one) => guardNotes(one, requirePerm)),
    relations: guarded(host.getRelations?.(), (one) => guardRelations(one, requirePerm)),
    files: guarded(host.getFiles?.(), (one) => guardFiles(one, requirePerm)),
    texts: guarded(host.getTexts?.(pluginId), (one) => guardTexts(one, requirePerm)),
    buffers: guarded(host.getBuffers?.(), (one) => guardBuffers(one, requireWrite, add)),
    vaults: host.getVaults?.(),
  })
}

function createUi(
  pluginId: string,
  host: PluginContextHost,
  mountView: PluginUi['mountView'],
): PluginUi {
  return {
    ...pickDefined<Omit<PluginUi, 'mountView'>>({
      workspace: host.getWorkspace?.(),
      window: host.getWindow?.(),
      commands: host.getCommands?.(),
      slots: host.getSlots?.(),
      notices: host.getNotices?.(pluginId),
      drops: host.drops,
    }),
    mountView,
  }
}

function createWork(
  pluginId: string,
  host: PluginContextHost,
  requirePerm: RequirePermission,
): PluginWork {
  return pickDefined<PluginWork>({
    agent: guarded(host.getAgent?.(), (one) => guardAgent(one, requirePerm)),
    tasks: host.getTasks?.(pluginId),
    backgroundTasks: host.getBackgroundTasks?.(),
  })
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
  const guard = { target, add }
  const slotView = createSlotViewRegistration(
    { pluginId, manifest: manifest && 'contributes' in manifest ? manifest : undefined },
    host,
    add,
  )

  return {
    plugin: {
      id: pluginId,
      target: EDITOR_TARGET,
      app: host.app,
      openExternal: createExternalOpener(host.openExternal),
      subscriptions,
      hasPermission: (p: string) => manifest?.permissions?.includes(p) ?? false,
    },
    register: createRegistrations(pluginId, host, { ...guard, slotView }),
    settings: createSettings(pluginId, host, add),
    events: createEvents(pluginId, host, { requirePerm, add }),
    tabs: createTabs(host),
    vault: createVault(pluginId, host, { requirePerm, add }),
    ui: createUi(pluginId, host, createMountView(host, guard)),
    work: createWork(pluginId, host, requirePerm),
  }
}
