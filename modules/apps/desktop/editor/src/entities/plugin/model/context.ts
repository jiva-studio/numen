/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '@numen/plugin-sdk'
import type { PluginContextHost } from './types'

export function createPluginContext(pluginId: string, host: PluginContextHost): PluginContext {
  const subscriptions: Disposable[] = []

  function addSubscription(disposable: Disposable): Disposable {
    subscriptions.push(disposable)
    return {
      dispose() {
        const index = subscriptions.indexOf(disposable)
        if (index !== -1) {
          subscriptions.splice(index, 1)
        }
        disposable.dispose()
      },
    }
  }

  return {
    pluginId,
    subscriptions,

    registerReader(registration: ReaderRegistration): Disposable {
      return addSubscription(host.registerReader(pluginId, registration))
    },

    registerTabKind(registration: TabKindRegistration): Disposable {
      return addSubscription(host.registerTabKind(pluginId, registration))
    },

    registerCommand(registration: CommandRegistration): Disposable {
      return addSubscription(host.registerCommand(pluginId, registration))
    },

    registerAgentProvider(registration: AgentProviderRegistration): Disposable {
      return addSubscription(host.registerAgentProvider(pluginId, registration))
    },

    getSetting<T = unknown>(key: string): T | undefined {
      return host.getSetting<T>(pluginId, key)
    },

    onSettingChange<T = unknown>(key: string, handler: (value: T) => void): Disposable {
      return addSubscription(host.onSettingChange<T>(pluginId, key, handler))
    },

    onVaultChange(handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable {
      return addSubscription(host.onVaultChange ? host.onVaultChange(handler) : { dispose: () => {} })
    },

    onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable {
      return addSubscription(host.onFileSelect ? host.onFileSelect(handler) : { dispose: () => {} })
    },

    emitVaultChange(event: VaultChangeEvent): void {
      host.emitVaultChange?.(event)
    },

    emitFileSelect(event: FileSelectEvent): void {
      host.emitFileSelect?.(event)
    },

    openTab(kind: string, at?: string): Promise<string> | void {
      return host.openTab?.(kind, at)
    },

    closeTab(id: string): void {
      host.closeTab?.(id)
    },
  }
}
