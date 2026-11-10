/**
 * The registrations and view mounting a plugin context offers, tracked for disposal.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  DropRegistration,
  PluginRegistrations,
  PluginUi,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotViewRegistration,
  TabKindRegistration,
  ViewRegistration,
} from '@numen/plugin-sdk'
import type { PluginContextHost } from '../types'
import type { PermissionTarget } from './permission'
import { requireMountAccess } from './viewPermissions'

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

/** The registrations of a plugin, tracked; the permissions they need come from the table of the services. */
export function createRegistrations(
  pluginId: string,
  host: PluginContextHost,
  deps: RegistrationDeps,
): PluginRegistrations {
  const { target, add, slotView } = deps
  const permissions = target.manifest?.permissions ?? []
  return {
    reader: (reg: ReaderRegistration) => add(host.registration.registerReader(pluginId, reg)),
    settingsView: (reg: SettingsViewRegistration) =>
      add(host.registration.registerSettingsView(pluginId, reg)),
    slotView,
    tabKind: <State extends object>(reg: TabKindRegistration<State>) =>
      add(host.registration.registerTabKind(pluginId, reg)),
    command: (reg: CommandRegistration) => add(host.registration.registerCommand(pluginId, reg)),
    agentProvider: (reg: AgentProviderRegistration) =>
      add(host.registration.registerAgentProvider(pluginId, reg)),
    view: (reg: ViewRegistration) => add(host.views.registerView(pluginId, reg, permissions)),
    dropHandler: (reg: DropRegistration) => add(host.views.registerDropHandler(pluginId, reg)),
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
