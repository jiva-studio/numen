import { type Manifest } from '@numen/core-client'
import type { SlotHost } from '@numen/plugin-sdk'
import { formatCommandId } from '../../commandId'
import { createPluginHealth } from '../../pluginHealth'
import type { DeclaredSlotEntry } from '../../slots'
import type { ContextUiHost, ContextWorkHost, HostCommands, PluginHostDeps } from '../types'

function findManifestCommandPermission(
  command: string,
  manifests: ReadonlyMap<string, Manifest>,
): string | undefined {
  for (const [pluginId, manifest] of manifests) {
    const match = manifest.contributes?.commands.find(
      (one) => formatCommandId(pluginId, one.id) === command,
    )
    if (match?.permission) return match.permission
  }
  return undefined
}

/** What the window a plugin draws in reads of the host: its slots, the manifests registered and the slot entries. */
interface UiServiceParts {
  readonly slots: SlotHost
  readonly manifests: ReadonlyMap<string, Manifest>
  readonly readSlotEntries: () => readonly DeclaredSlotEntry[] | undefined
}

const isListedIn = (entries: readonly DeclaredSlotEntry[], command: string): boolean =>
  entries.some(
    (one) =>
      one.command !== undefined &&
      (one.command.includes(':') ? one.command : formatCommandId(one.pluginId, one.command)) ===
        command,
  )

const findCommandPermission = (
  commands: HostCommands,
  manifests: ReadonlyMap<string, Manifest>,
  command: string,
): string | undefined =>
  (commands.getCommandDeclarations() ?? []).find(
    (one) => formatCommandId(one.pluginId, one.id) === command,
  )?.permission ||
  findManifestCommandPermission(command, manifests) ||
  undefined

/** The window a plugin draws in as a plugin context reads it, each service read at the moment a plugin asks. */
export function createUiServiceBridge(
  { window, commands }: Pick<PluginHostDeps, 'window' | 'commands'>,
  { slots, manifests, readSlotEntries }: UiServiceParts,
): ContextUiHost {
  return {
    app: window.app,
    target: window.target,
    openExternal: window.openExternal,
    getWorkspace: window.getWorkspace,
    getCommands: window.getCommands,
    commandAccess: {
      isListed: (command) => isListedIn(readSlotEntries() ?? [], command),
      getPermission: (command) => findCommandPermission(commands, manifests, command),
    },
    getWindow: window.getWindow,
    getVaultFlags: window.getVaultFlags,
    getNotices: window.getNotices,
    getPicker: window.getPicker,
    getDialogs: window.getDialogs,
    getSlots: () => slots,
  }
}

/** The background work of the window as a plugin context reads it, each service read at the moment a plugin asks. */
export function createWorkServiceBridge({ work }: Pick<PluginHostDeps, 'work'>): ContextWorkHost {
  return {
    getTasks: work.getTasks,
    getBackgroundTasks: work.getBackgroundTasks,
    getProcesses: work.getProcesses,
    getLogs: work.getLogs,
    getHealth: (pluginId) => createPluginHealth(work.health, pluginId),
    getConnections: work.getConnections,
  }
}
