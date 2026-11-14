/**
 * The calls of the core that the plugin host makes for plugins: the texts they read, and the
 * operations their rows of connections carry out.
 */
import { createPluginConnections, createPluginTexts } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'

/** The texts and the connections a plugin reaches, over the clients of the window. */
export const createPluginHostApi = ({
  texts,
  connectionCalls,
}: Pick<WindowClients, 'texts' | 'connectionCalls'>) => ({
  /** The texts of the vault, as the plugin of that id reads them. */
  getPluginTexts: (pluginId: string): ReturnType<typeof createPluginTexts> =>
    createPluginTexts(texts, pluginId),

  /**
   * The connections a plugin lists in its rows, as the plugin of that id calls them.
   * `subscribeChanges` notifies them of a change of the rows.
   */
  getPluginConnections: (
    pluginId: string,
    subscribeChanges: (handler: () => void) => () => void,
  ): ReturnType<typeof createPluginConnections> =>
    createPluginConnections(connectionCalls, pluginId, subscribeChanges),
})

export type PluginHostApi = ReturnType<typeof createPluginHostApi>
