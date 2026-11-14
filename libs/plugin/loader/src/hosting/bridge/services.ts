import { ReportedError, type Manifest } from '@numen/core-client'
import type { NoticeHost, SlotHost } from '@numen/plugin-sdk'
import { createAppInfo } from '../../appInfo'
import { formatCommandId } from '../../commandId'
import { createPluginHealth } from '../../pluginHealth'
import type { DeclaredSlotEntry } from '../../slots'
import type {
  ContextUiHost,
  ContextVaultHost,
  ContextWorkHost,
  HostCommands,
  PluginHostDeps,
} from '../types'

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
export function createVaultServiceBridge({
  vault,
}: Pick<PluginHostDeps, 'vault'>): ContextVaultHost {
  return {
    getNotes: () => vault?.getNotes?.(),
    getRelations: () => vault?.getRelations?.(),
    getFiles: () => vault?.getFiles?.(),
    getTexts: (pluginId) => vault?.getTexts?.(pluginId),
    getBuffers: () => vault?.getBuffers?.(),
    getVaults: () => vault?.getVaults?.(),
  }
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
  commands: HostCommands | undefined,
  manifests: ReadonlyMap<string, Manifest>,
  command: string,
): string | undefined =>
  (commands?.getCommandDeclarations?.() ?? []).find(
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
    app: window.app ?? createAppInfo({ getPlatform: () => 'linux' }),
    target: window.target,
    openExternal: (url) =>
      window.openExternal
        ? window.openExternal(url)
        : Promise.reject(new ReportedError('This window cannot open web links')),
    getWorkspace: () => window.getWorkspace?.(),
    getCommands: () => window.getCommands?.(),
    isCommandListed: (command) => isListedIn(readSlotEntries() ?? [], command),
    getCommandPermission: (command) => findCommandPermission(commands, manifests, command),
    getWindow: () => window.getWindow?.(),
    getNotices: (pluginId) => window.getNotices?.(pluginId) ?? DISCARDING_NOTICES,
    getSlots: () => slots,
  }
}

/** The background work of the window as a plugin context reads it, each service read at the moment a plugin asks. */
export function createWorkServiceBridge({ work }: Pick<PluginHostDeps, 'work'>): ContextWorkHost {
  return {
    getTasks: (pluginId) => work?.getTasks?.(pluginId),
    getBackgroundTasks: () => work?.getBackgroundTasks?.(),
    getProcesses: () => work?.getProcesses?.(),
    getHealth: (pluginId) => work?.health && createPluginHealth(work.health, pluginId),
    getConnections: (pluginId) => work?.getConnections?.(pluginId),
  }
}
