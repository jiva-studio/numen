/**
 * Host implementation of PluginContext and HostServices conforming to @numen/plugin-sdk.
 */
import type {
  BufferHost,
  ConnectionHost,
  Disposable,
  FileBuffers,
  FileSelectEvent,
  HostServices,
  JsonValue,
  PluginContext,
  PluginEvents,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import { isUnsavedBuffersError, UnsavedBuffersError } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createExternalOpener } from '../externalOpener'
import { limitCommands } from './guards/commandRun'
import { checkPermission, type PermissionTarget, type RequirePermission } from './guards/permission'
import {
  createMountView,
  createRegistrations,
  type RegistrationGuard,
  type Track,
} from './guards/registrations'
import { guardService, SERVICE_RULES } from './guards/servicePermissions'
import { createSubscriptionList, type SubscriptionList } from './guards/subscriptions'
import { createSlotViewRegistration } from './guards/slotContext'
import type { PluginParts } from '../host'
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

function nameUnsaved(pluginId: string, flush: () => Promise<void>): () => Promise<void> {
  return async () => {
    try {
      await flush()
    } catch (error) {
      if (!isUnsavedBuffersError(error)) throw error
      throw new UnsavedBuffersError(error.unsaved, pluginId)
    }
  }
}

function createBuffers(pluginId: string, buffers: BufferHost, add: Track): BufferHost {
  return {
    track: (open) =>
      add(
        buffers.track(
          Object.assign(Object.create(open) as FileBuffers, {
            flush: nameUnsaved(pluginId, open.flush),
          }),
        ),
      ),
    addFlushHandler: (handler) => add(buffers.addFlushHandler(nameUnsaved(pluginId, handler))),
  }
}

function createSettings(pluginId: string, host: PluginContextHost, add: Track): PluginSettings {
  return {
    get: <T extends JsonValue = JsonValue>(key: string) =>
      host.settings.getSetting<T>(pluginId, key),
    onChange: <T extends JsonValue = JsonValue>(key: string, h: (v: T) => void) =>
      add(host.settings.onSettingChange<T>(pluginId, key, h)),
  }
}

function createEvents(pluginId: string, host: PluginContextHost, add: Track): PluginEvents {
  return {
    onVaultChange: (h: (e: VaultChangeEvent) => void | Promise<void>) =>
      add(host.events.onVaultChange(pluginId, h)),
    onFileSelect: (h: (e: FileSelectEvent) => void | Promise<void>) =>
      add(host.events.onFileSelect(pluginId, h)),
    emitVaultChange: (e: VaultChangeEvent) => host.events.emitVaultChange(pluginId, e),
    emitFileSelect: (e: FileSelectEvent) => host.events.emitFileSelect(pluginId, e),
  }
}

function createTabs(host: PluginContextHost): PluginTabs {
  return {
    open: (kind: string, targetPath?: string) => host.tabs.openTab(kind, targetPath),
    navigate: (kind: string, targetPath: string) => host.tabs.navigateTab(kind, targetPath),
    close: (id: string) => host.tabs.closeTab(id),
  }
}

function pickDefinedParts<T extends object>(parts: { [K in keyof T]: T[K] | undefined }): T {
  const picked: Partial<T> = {}
  for (const key of Object.keys(parts) as (keyof T)[]) {
    const value = parts[key]
    if (value !== undefined) picked[key] = value
  }
  return picked as T
}

function createVault(pluginId: string, host: PluginContextHost, add: Track): PluginVault {
  const { vault } = host
  const buffers = vault.getBuffers()
  return pickDefinedParts<PluginVault>({
    notes: vault.getNotes(),
    relations: vault.getRelations(),
    files: vault.getFiles(),
    texts: vault.getTexts(pluginId),
    buffers: buffers && createBuffers(pluginId, buffers, add),
    vaults: vault.getVaults(),
  })
}

function createUi(
  pluginId: string,
  host: PluginContextHost,
  check: RequirePermission,
): Omit<PluginUi, 'mountView'> {
  const { ui, views } = host
  const commands = ui.getCommands()
  return pickDefinedParts<Omit<PluginUi, 'mountView'>>({
    workspace: ui.getWorkspace(),
    window: ui.getWindow(),
    commands:
      commands &&
      limitCommands(commands, {
        pluginId,
        isListed: (one) => ui.isCommandListed(one),
        getPermission: (one) => ui.getCommandPermission?.(one),
        checkPermission: check,
      }),
    drops: views.drops,
    slots: ui.getSlots(),
    notices: ui.getNotices(pluginId),
  })
}

function createUiService(
  pluginId: string,
  host: PluginContextHost,
  guards: { readonly guard: RegistrationGuard; readonly check: RequirePermission },
): PluginUi {
  const ui = guardService(createUi(pluginId, host, guards.check), SERVICE_RULES.ui, {
    check: guards.check,
  })
  return pickDefinedParts<PluginUi>({
    workspace: ui.workspace,
    window: ui.window,
    commands: ui.commands,
    drops: ui.drops,
    slots: ui.slots,
    notices: ui.notices,
    mountView: createMountView(host, guards.guard),
  })
}

function trackConnections(
  connections: ConnectionHost | undefined,
  add: Track,
): ConnectionHost | undefined {
  const onChange = connections?.onChange?.bind(connections)
  if (!connections || !onChange) return connections
  return { ...connections, onChange: (handler) => add(onChange(handler)) }
}

function createWork(pluginId: string, host: PluginContextHost, add: Track): PluginWork {
  const { work } = host
  return pickDefinedParts<PluginWork>({
    tasks: work.getTasks(pluginId),
    backgroundTasks: work.getBackgroundTasks(),
    processes: work.getProcesses(),
    connections: trackConnections(work.getConnections(pluginId), add),
  })
}

/** What both factories build on: the plugin, its permission check and its subscriptions. */
interface PartsScope {
  readonly target: PermissionTarget
  readonly subscriptions: SubscriptionList
  readonly add: Track
  readonly check: RequirePermission
}

/**
 * The context the plugin owns: its identity, its registrations, its settings and its health. Every
 * service in it is built from the table of SERVICE_RULES: a member the table does not list is not
 * reachable, and a member that lists a permission throws, or rejects, where the manifest of the
 * plugin does not declare it.
 */
function createPluginContext(scope: PartsScope, host: PluginContextHost): PluginContext {
  const { target, add, check } = scope
  const { pluginId, manifest } = target
  const slotView = createSlotViewRegistration(
    { pluginId, manifest: manifest && 'contributes' in manifest ? manifest : undefined },
    host,
    add,
  )
  const rules = SERVICE_RULES
  const health = host.work.getHealth(pluginId)
  return pickDefinedParts<PluginContext>({
    identity: {
      id: pluginId,
      target: host.ui.target,
      app: host.ui.app,
      openExternal: guardService(
        { openExternal: createExternalOpener(host.ui.openExternal) },
        rules.identity,
        { check },
      ).openExternal,
      subscriptions: scope.subscriptions.view,
      hasPermission: (p: string) => manifest?.permissions?.includes(p) ?? false,
    },
    register: guardService(
      createRegistrations(pluginId, host, { target, add, slotView }),
      rules.register,
      { check },
    ),
    settings: guardService(createSettings(pluginId, host, add), rules.settings, { check }),
    health: health && guardService(health, rules.health, { check }),
  })
}

/**
 * The services the host lends the plugin: the vault, the tabs, the window, the background work and
 * the events, guarded by SERVICE_RULES as the plugin context is. What the plugin is handed through
 * them is tracked in the subscriptions of the plugin.
 */
function createHostServices(scope: PartsScope, host: PluginContextHost): HostServices {
  const { target, add, check } = scope
  const { pluginId } = target
  const rules = SERVICE_RULES
  return {
    vault: guardService(createVault(pluginId, host, add), rules.vault, { check }),
    tabs: guardService(createTabs(host), rules.tabs, { check }),
    ui: createUiService(pluginId, host, { guard: scope, check }),
    work: guardService(createWork(pluginId, host, add), rules.work, { check }),
    events: guardService(createEvents(pluginId, host, add), rules.events, { check }),
  }
}

/** The arguments of `activate`: the context the plugin owns and the services the host lends it. */
export function createPluginParts(
  pluginId: string,
  host: PluginContextHost,
  manifest?: Manifest | { permissions?: readonly string[] },
): PluginParts {
  const subscriptions = createSubscriptionList()
  const target: PermissionTarget = { pluginId, manifest }
  const scope: PartsScope = {
    target,
    subscriptions,
    add: (d: Disposable) => trackSubscription(subscriptions, d),
    check: (p, op) => checkPermission(target, p, op),
  }
  return { plugin: createPluginContext(scope, host), host: createHostServices(scope, host) }
}
