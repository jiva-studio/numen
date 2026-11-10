/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  BufferHost,
  Disposable,
  FileSelectEvent,
  PluginContext,
  JsonValue,
  PluginEvents,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createExternalOpener } from '../externalOpener'
import { limitAgentSettings } from './guards/agentSettings'
import { checkPermission, type PermissionTarget, type RequirePermission } from './guards/permission'
import { createMountView, createRegistrations, type Track } from './guards/registrations'
import { guardService, SERVICE_RULES } from './guards/servicePermissions'
import { createSubscriptionList, type SubscriptionList } from './guards/subscriptions'
import { createSlotViewRegistration } from './guards/slotContext'
import type { PluginContextHost } from './types'

function trackSubscription(subscriptions: SubscriptionList, disposable: Disposable): Disposable {
  subscriptions.add(disposable)
  return {
    dispose() {
      subscriptions.remove(disposable)
      disposable.dispose()
    },
  }
}

function createBuffers(buffers: BufferHost, add: Track): BufferHost {
  return {
    track: (open) => add(buffers.track(open)),
    addFlushHandler: (handler) => add(buffers.addFlushHandler(handler)),
  }
}

/** The target the editor activates its plugins in. */
const EDITOR_TARGET = 'desktop.editor'

function createSettings(pluginId: string, host: PluginContextHost, add: Track): PluginSettings {
  return {
    get: <T extends JsonValue = JsonValue>(key: string) => host.getSetting<T>(pluginId, key),
    onChange: <T extends JsonValue = JsonValue>(key: string, h: (v: T) => void) =>
      add(host.onSettingChange<T>(pluginId, key, h)),
  }
}

function createEvents(pluginId: string, host: PluginContextHost, add: Track): PluginEvents {
  return {
    onVaultChange: (h: (e: VaultChangeEvent) => void | Promise<void>) =>
      add(host.onVaultChange(pluginId, h)),
    onFileSelect: (h: (e: FileSelectEvent) => void | Promise<void>) =>
      add(host.onFileSelect(pluginId, h)),
    emitVaultChange: (e: VaultChangeEvent) => host.emitVaultChange(pluginId, e),
    emitFileSelect: (e: FileSelectEvent) => host.emitFileSelect(pluginId, e),
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

function pickDefined<T extends object>(parts: { [K in keyof T]: T[K] | undefined }): T {
  const picked: Partial<T> = {}
  for (const key of Object.keys(parts) as (keyof T)[]) {
    const value = parts[key]
    if (value !== undefined) picked[key] = value
  }
  return picked as T
}

function createVault(pluginId: string, host: PluginContextHost, add: Track): PluginVault {
  const buffers = host.getBuffers?.()
  return pickDefined<PluginVault>({
    notes: host.getNotes?.(),
    relations: host.getRelations?.(),
    files: host.getFiles?.(),
    texts: host.getTexts?.(pluginId),
    buffers: buffers && createBuffers(buffers, add),
    vaults: host.getVaults?.(),
  })
}

function createUi(pluginId: string, host: PluginContextHost): Omit<PluginUi, 'mountView'> {
  return {
    ...pickDefined<Omit<PluginUi, 'mountView' | 'slots' | 'notices'>>({
      workspace: host.getWorkspace?.(),
      window: host.getWindow?.(),
      commands: host.getCommands?.(),
      drops: host.drops,
    }),
    slots: host.getSlots(),
    notices: host.getNotices(pluginId),
  }
}

function createWork(pluginId: string, host: PluginContextHost): PluginWork {
  const agent = host.getAgent?.()
  return pickDefined<PluginWork>({
    agent: agent && limitAgentSettings(agent),
    tasks: host.getTasks?.(pluginId),
    backgroundTasks: host.getBackgroundTasks?.(),
  })
}

/**
 * The context of a plugin. Every service in it is built from the table of SERVICE_RULES: a member
 * the table does not list is not reachable, and a member that lists a permission throws, or rejects,
 * where the manifest of the plugin does not declare it.
 */
export function createPluginContext(
  pluginId: string,
  host: PluginContextHost,
  manifest?: Manifest | { permissions?: readonly string[] },
): PluginContext {
  const subscriptions = createSubscriptionList()
  const add = (d: Disposable) => trackSubscription(subscriptions, d)
  const target: PermissionTarget = { pluginId, manifest }
  const check: RequirePermission = (p, op) => checkPermission(target, p, op)
  const registration = { target, add }
  const slotView = createSlotViewRegistration(
    { pluginId, manifest: manifest && 'contributes' in manifest ? manifest : undefined },
    host,
    add,
  )
  const rules = SERVICE_RULES

  return {
    plugin: {
      id: pluginId,
      target: host.target ?? EDITOR_TARGET,
      app: host.app,
      openExternal: createExternalOpener(host.openExternal),
      subscriptions: subscriptions.view,
      hasPermission: (p: string) => manifest?.permissions?.includes(p) ?? false,
    },
    register: guardService(
      createRegistrations(pluginId, host, { ...registration, slotView }),
      rules.register,
      { check },
    ),
    settings: guardService(createSettings(pluginId, host, add), rules.settings, { check }),
    events: guardService(createEvents(pluginId, host, add), rules.events, { check }),
    tabs: guardService(createTabs(host), rules.tabs, { check }),
    vault: guardService(createVault(pluginId, host, add), rules.vault, { check }),
    ui: {
      ...guardService(createUi(pluginId, host), rules.ui, { check }),
      mountView: createMountView(host, registration),
    },
    work: guardService(createWork(pluginId, host), rules.work, { check }),
  }
}
