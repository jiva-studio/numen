/**
 * The registrations and view mounting a plugin context offers, tracked for disposal.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  DropRegistration,
  ReaderRegistration,
  SettingsViewRegistration,
  TabKindRegistration,
  ViewRegistration,
} from '@numen/plugin-sdk'
import type { PluginContextHost } from './types'
import { getViewPermissions } from './viewPermissions'

export type Track = (disposable: Disposable) => Disposable
export type RequirePermission = (permission: string, operation: string) => void

/** What the registrations of a plugin are tracked and guarded with. */
export interface RegistrationGuard {
  readonly requirePerm: RequirePermission
  readonly add: Track
}

export function createRegistrations(
  pluginId: string,
  host: PluginContextHost,
  guard: RegistrationGuard,
) {
  const { requirePerm, add } = guard
  return {
    registerReader: (reg: ReaderRegistration) => add(host.registerReader(pluginId, reg)),
    registerSettingsView: (reg: SettingsViewRegistration) =>
      add(host.registerSettingsView(pluginId, reg)),
    registerTabKind: <State>(reg: TabKindRegistration<State>) =>
      add(host.registerTabKind(pluginId, reg)),
    registerCommand: (reg: CommandRegistration) => add(host.registerCommand(pluginId, reg)),
    registerAgentProvider: (reg: AgentProviderRegistration) =>
      add(host.registerAgentProvider(pluginId, reg)),
    registerView: (reg: ViewRegistration) => add(host.registerView(pluginId, reg)),
    mountView: (capability: string, element: HTMLElement, context: unknown) => {
      for (const permission of getViewPermissions(capability)) requirePerm(permission, 'mountView')
      const view = host.mountView(capability, element, context)
      return { focus: () => view.focus(), ...add(view) }
    },
    registerDropHandler: (reg: DropRegistration) => add(host.registerDropHandler(pluginId, reg)),
  }
}
