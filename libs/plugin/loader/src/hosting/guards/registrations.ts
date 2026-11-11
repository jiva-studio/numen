/**
 * The registrations and view mounting a plugin context offers, tracked for disposal.
 */
import {
  isSharedCommand,
  type AgentProviderRegistration,
  type CommandRegistration,
  type Disposable,
  type DropRegistration,
  type PluginRegistrations,
  type PluginUi,
  type ReaderRegistration,
  type SettingsViewRegistration,
  type SlotViewRegistration,
  type TabKindRegistration,
  type ViewRegistration,
} from '@numen/plugin-sdk'
import type { PluginContextHost } from '../types'
import {
  requireDeclaredAgentProvider,
  requireDeclaredCommand,
  requireDeclaredDropTypes,
  requireDeclaredReader,
  requireDeclaredSettingsView,
  requireDeclaredTabKind,
} from './declarations'
import type { PermissionTarget } from './permission'
import { requireMountAccess, requireViewRegistration } from './viewPermissions'

export type Track = (disposable: Disposable) => Disposable

/** What the registrations of a plugin are tracked and guarded with. */
export interface RegistrationGuard {
  readonly target: PermissionTarget
  readonly add: Track
}

/** The guard of the registrations, and how the views of the plugin are registered in the slots. */
export interface RegistrationDeps extends RegistrationGuard {
  readonly slotView: (registration: SlotViewRegistration) => Disposable
}

function takeSnapshot<Registration extends object>(reg: Registration): Registration {
  const snapshot: Registration & { placement?: object } = { ...reg }
  if (snapshot.placement) snapshot.placement = { ...snapshot.placement }
  return snapshot
}

/**
 * The registrations of a plugin, tracked; the permissions they need come from the table of the services.
 * Each registration is copied once before it is checked, so a getter on it cannot answer the check and the
 * host differently; the tab placement is the one nested object the host reads.
 */
export function createRegistrations(
  pluginId: string,
  host: PluginContextHost,
  deps: RegistrationDeps,
): PluginRegistrations {
  const { target, add, slotView } = deps
  const permissions = target.manifest?.permissions ?? []
  return {
    reader: (reg: ReaderRegistration) => {
      const snapshot = takeSnapshot(reg)
      requireDeclaredReader(target, snapshot.id)
      return add(host.registration.registerReader(pluginId, snapshot))
    },
    settingsView: (reg: SettingsViewRegistration) => {
      const snapshot = takeSnapshot(reg)
      requireDeclaredSettingsView(target, snapshot.id)
      return add(host.registration.registerSettingsView(pluginId, snapshot))
    },
    slotView,
    tabKind: <State extends object>(reg: TabKindRegistration<State>) => {
      const snapshot = takeSnapshot(reg)
      requireDeclaredTabKind(target, snapshot.kind)
      return add(host.registration.registerTabKind(pluginId, snapshot))
    },
    command: (reg: CommandRegistration) => {
      const snapshot = takeSnapshot(reg)
      requireDeclaredCommand(target, snapshot.id, isSharedCommand(snapshot.id))
      return add(host.registration.registerCommand(pluginId, snapshot))
    },
    agentProvider: (reg: AgentProviderRegistration) => {
      const snapshot = takeSnapshot(reg)
      requireDeclaredAgentProvider(target, snapshot.id)
      return add(host.registration.registerAgentProvider(pluginId, snapshot))
    },
    view: (reg: ViewRegistration) => {
      // Read once: a getter on the registration cannot answer the check and the registry differently.
      const { capability, component } = reg
      requireViewRegistration(capability, target)
      return add(host.views.registerView(pluginId, { capability, component }, permissions))
    },
    dropHandler: (reg: DropRegistration) => {
      // Read once: the types checked are the types registered, and a later change to either is not seen.
      const { types, drop } = reg
      const declared = requireDeclaredDropTypes(target, types)
      return add(host.views.registerDropHandler(pluginId, { types: declared, drop }))
    },
  }
}

/** Draws the view a plugin declares into an element, after the access check of the plugin. */
export function createMountView(
  host: PluginContextHost,
  guard: RegistrationGuard,
): PluginUi['mountView'] {
  const { target, add } = guard
  return (capability, element, context) => {
    const access = requireMountAccess(capability, target)
    const view = host.views.mountView(capability, element, context, access)
    return { focus: () => view.focus(), ...add(view) }
  }
}
