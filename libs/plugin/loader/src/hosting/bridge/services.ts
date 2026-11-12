import type { Manifest } from '@numen/core-client'
import type { NoticeHost, SlotHost } from '@numen/plugin-sdk'
import { formatCommandId } from '../../commandId'
import { createPluginHealth } from '../../pluginHealth'
import type { DeclaredSlotEntry } from '../../slots'
import type { ContextVaultHost, ContextWindowHost, PluginHostDeps } from '../types'

/** The notices of a window that shows none: what a plugin reports is dropped. */
const DISCARDING_NOTICES: NoticeHost = { show: () => undefined, setTasks: () => undefined }

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

/** The vault services of the window as a plugin context reads them, each read at the moment a plugin asks. */
export function createVaultServiceBridge(deps: PluginHostDeps): ContextVaultHost {
  return {
    getNotes: () => deps.getNotes?.(),
    getRelations: () => deps.getRelations?.(),
    getFiles: () => deps.getFiles?.(),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
    getBuffers: () => deps.getBuffers?.(),
    getVaults: () => deps.getVaults?.(),
  }
}

/** What the window services read of the host: its slots, the manifests registered and the slot entries. */
interface WindowServiceParts {
  readonly slots: SlotHost
  readonly manifests: ReadonlyMap<string, Manifest>
  readonly readSlotEntries: () => readonly DeclaredSlotEntry[] | undefined
}

/** The services of the window itself as a plugin context reads them, each read at the moment a plugin asks. */
export function createWindowServiceBridge(
  deps: PluginHostDeps,
  { slots, manifests, readSlotEntries }: WindowServiceParts,
): ContextWindowHost {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getCommands: () => deps.getCommands?.(),
    isCommandListed: (command) =>
      (readSlotEntries() ?? []).some(
        (one) =>
          one.command !== undefined &&
          (one.command.includes(':') ? one.command : formatCommandId(one.pluginId, one.command)) ===
            command,
      ),
    getCommandPermission: (command) =>
      (deps.getCommandDeclarations?.() ?? []).find(
        (one) => formatCommandId(one.pluginId, one.id) === command,
      )?.permission ||
      findManifestCommandPermission(command, manifests) ||
      undefined,
    getWindow: () => deps.getWindow?.(),
    getNotices: (pluginId) => deps.getNotices?.(pluginId) ?? DISCARDING_NOTICES,
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getBackgroundTasks: () => deps.getBackgroundTasks?.(),
    getHealth: (pluginId) => deps.health && createPluginHealth(deps.health, pluginId),
    getConnections: (pluginId) => deps.getConnections?.(pluginId),
    getSlots: () => slots,
  }
}
