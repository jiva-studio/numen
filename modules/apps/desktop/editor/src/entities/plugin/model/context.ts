/**
 * Host implementation of PluginContext conforming to @numen/plugin-sdk.
 */
import type {
  AgentProviderRegistration,
  CommandRegistration,
  Disposable,
  PluginContext,
  ReaderRegistration,
  TabKindRegistration,
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
  }
}
