/**
 * The registrations and view mounting a plugin context offers, tracked for disposal.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  DropRegistration,
  MountedView,
  PluginRegistrations,
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
export type RequirePermission = (permission: string, operation: string) => void

/** What the registrations of a plugin are tracked and guarded with. */
export interface RegistrationGuard {
  readonly target: PermissionTarget
  readonly add: Track
}

export function createRegistrations(
  pluginId: string,
  host: PluginContextHost,
  guard: RegistrationGuard,
  slotView: (registration: SlotViewRegistration) => Disposable,
): PluginRegistrations {
  const { target, add } = guard
  const permissions = target.manifest?.permissions ?? []
  return {
    reader: (reg: ReaderRegistration) => add(host.registerReader(pluginId, reg)),
    settingsView: (reg: SettingsViewRegistration) => add(host.registerSettingsView(pluginId, reg)),
    slotView,
    tabKind: <State>(reg: TabKindRegistration<State>) => add(host.registerTabKind(pluginId, reg)),
    command: (reg: CommandRegistration) => add(host.registerCommand(pluginId, reg)),
    agentProvider: (reg: AgentProviderRegistration) =>
      add(host.registerAgentProvider(pluginId, reg)),
    view: (reg: ViewRegistration) => add(host.registerView(pluginId, reg, permissions)),
    dropHandler: (reg: DropRegistration) => add(host.registerDropHandler(pluginId, reg)),
  }
}

/** Draws the view a plugin declares into an element, after the access check of the plugin. */
export function createMountView(
  host: PluginContextHost,
  guard: RegistrationGuard,
): (capability: string, element: HTMLElement, context: unknown) => MountedView {
  const { target, add } = guard
  return (capability, element, context) => {
    const access = requireMountAccess(capability, target)
    const view = host.mountView(capability, element, context, access)
    return { focus: () => view.focus(), ...add(view) }
  }
}
