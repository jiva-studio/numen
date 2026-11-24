/**
 * Host implementation of PluginContext and HostServices conforming to @numen/plugin-sdk.
 */
import type {
  BufferHost,
  ConnectionHost,
  Disposable,
  EditorExtensionHost,
  EditorSettings,
  FileBuffers,
  FileSelectEvent,
  HostServices,
  JsonValue,
  LogHost,
  PluginContext,
  PluginEvents,
  PluginLicenses,
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
import { requireDeclaredLicense } from './guards/declarations'
import { checkPermission, type PermissionTarget, type RequirePermission } from './guards/permission'
import {
  createMountView,
  createRegistrations,
  type RegistrationGuard,
  type Track,
} from './guards/registrations'
import { guardService } from './guards/guardService'
import { SERVICE_RULES } from './guards/servicePermissions'
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

function wrapUnsavedFlush(pluginId: string, flush: () => Promise<void>): () => Promise<void> {
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
            flush: wrapUnsavedFlush(pluginId, open.flush),
          }),
        ),
      ),
    addFlushHandler: (handler) => add(buffers.addFlushHandler(wrapUnsavedFlush(pluginId, handler))),
  }
}

function createSettings(pluginId: string, host: PluginContextHost, add: Track): PluginSettings {
  return {
    get: <T extends JsonValue = JsonValue>(key: string) =>
      host.settings.getSetting<T>(pluginId, key),
    onChange: <T extends JsonValue = JsonValue>(key: string, h: (v: T) => void) =>
      add(host.settings.watchSetting<T>(pluginId, key, h)),
    getEditor: () => host.settings.getEditorSettings(),
    onEditorChange: (h: (settings: EditorSettings) => void) =>
      add(host.settings.watchEditorSettings(pluginId, h)),
  }
}

function createLicenses(
  target: PermissionTarget,
  host: PluginContextHost,
  add: Track,
): PluginLicenses {
  const { pluginId } = target
  return {
    has: (licenseId: string) => {
      requireDeclaredLicense(target, licenseId)
      return host.settings.hasLicense(pluginId, licenseId)
    },
    getKey: (licenseId: string) => {
      requireDeclaredLicense(target, licenseId)
      return host.settings.getLicenseKey(pluginId, licenseId)
    },
    onChange: (licenseId: string, handler: (isActive: boolean) => void) => {
      requireDeclaredLicense(target, licenseId)
      return add(host.settings.watchLicense(pluginId, licenseId, handler))
    },
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

function trackEditorExtensions(extensions: EditorExtensionHost, add: Track): EditorExtensionHost {
  return { ...extensions, onChange: (handler) => add(extensions.onChange(handler)) }
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
    vaultFlags: ui.forPlugin.getVaultFlags(pluginId),
    commands:
      commands &&
      limitCommands(commands, {
        pluginId,
        isListed: (one) => ui.commandAccess.isListed(one),
        getPermission: (one) => ui.commandAccess.getPermission?.(one),
        checkPermission: check,
      }),
    drops: views.drops,
    slots: ui.getSlots(),
    notices: ui.forPlugin.getNotices(pluginId),
    picker: ui.prompts.getPicker(),
    dialogs: ui.prompts.getDialogs(),
  })
}

function createUiService(
  pluginId: string,
  host: PluginContextHost,
  check: RequirePermission,
): Omit<PluginUi, 'mountView'> {
  const ui = guardService(createUi(pluginId, host, check), SERVICE_RULES.ui, { check })
  return pickDefinedParts<Omit<PluginUi, 'mountView'>>({
    workspace: ui.workspace,
    window: ui.window,
    vaultFlags: ui.vaultFlags,
    commands: ui.commands,
    drops: ui.drops,
    slots: ui.slots,
    notices: ui.notices,
    picker: ui.picker,
    dialogs: ui.dialogs,
  })
}

/** The window a plugin draws in: the guarded services of the window and the view mounter of the plugin. */
function addMountView(ui: Omit<PluginUi, 'mountView'>, mountView: PluginUi['mountView']): PluginUi {
  return { ...ui, mountView }
}

function trackConnections(
  connections: ConnectionHost | undefined,
  add: Track,
): ConnectionHost | undefined {
  const onChange = connections?.onChange?.bind(connections)
  if (!connections || !onChange) return connections
  return { ...connections, onChange: (handler) => add(onChange(handler)) }
}

function trackLogs(logs: LogHost | undefined, add: Track): LogHost | undefined {
  if (!logs) return undefined
  return { ...logs, watch: (handler) => add(logs.watch(handler)) }
}

function createWork(pluginId: string, host: PluginContextHost, add: Track): PluginWork {
  const { work } = host
  return pickDefinedParts<PluginWork>({
    tasks: work.getTasks(pluginId),
    backgroundTasks: work.getBackgroundTasks(),
    processes: work.getProcesses(),
    logs: trackLogs(work.getLogs(pluginId), add),
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
    licenses: guardService(createLicenses(target, host, add), rules.licenses, { check }),
    health: health && guardService(health, rules.health, { check }),
  })
}

function createHostServices(scope: PartsScope, host: PluginContextHost): HostServices {
  const { target, add, check } = scope
  const { pluginId } = target
  const rules = SERVICE_RULES
  return {
    vault: guardService(createVault(pluginId, host, add), rules.vault, { check }),
    tabs: guardService(createTabs(host), rules.tabs, { check }),
    ui: addMountView(createUiService(pluginId, host, check), createMountView(host, scope)),
    work: guardService(createWork(pluginId, host, add), rules.work, { check }),
    events: guardService(createEvents(pluginId, host, add), rules.events, { check }),
    editor: guardService(trackEditorExtensions(host.views.editorExtensions, add), rules.editor, {
      check,
    }),
  }
}

/** The parts a plugin is given, with the list of its subscriptions that only the host reaches. */
export interface HostedParts extends PluginParts {
  readonly subscriptions: SubscriptionList
}

/**
 * The arguments of `activate`: the context the plugin owns (identity, registrations, settings,
 * health) and the services the host lends it (vault, tabs, window, background work, events). Every
 * service in both is built from the table of SERVICE_RULES: a member the table does not list is not
 * reachable, and a member that lists a permission throws, or rejects, where the manifest of the
 * plugin does not declare it. What the plugin is handed through them is tracked in its
 * subscriptions.
 */
export function createPluginParts(
  pluginId: string,
  host: PluginContextHost,
  manifest?: Manifest | { permissions?: readonly string[] },
): HostedParts {
  const subscriptions = createSubscriptionList()
  const target: PermissionTarget = { pluginId, manifest }
  const scope: PartsScope = {
    target,
    subscriptions,
    add: (d: Disposable) => trackSubscription(subscriptions, d),
    check: (p, op) => checkPermission(target, p, op),
  }
  return {
    plugin: createPluginContext(scope, host),
    host: createHostServices(scope, host),
    subscriptions,
  }
}
