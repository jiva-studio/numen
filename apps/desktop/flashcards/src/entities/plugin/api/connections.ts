import { createPluginConnections } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'

/**
 * The connections a plugin lists in its rows, as the plugin of that id calls them over
 * `connectionCalls`.
 */
export const createPluginConnectionFinder =
  (connectionCalls: WindowClients['connectionCalls']) =>
  (pluginId: string): ReturnType<typeof createPluginConnections> =>
    createPluginConnections(connectionCalls, pluginId)
