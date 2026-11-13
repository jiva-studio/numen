/**
 * The calls of the core that the plugin host makes for plugins: the texts they read, and the
 * operations their rows of connections carry out.
 */
import { createPluginConnections, createPluginTexts } from '@numen/plugin-loader'
import { connectionCalls, texts } from '@/shared/api/clients'

/** The texts of the vault, as the plugin of that id reads them. */
export const getPluginTexts = (pluginId: string): ReturnType<typeof createPluginTexts> =>
  createPluginTexts(texts, pluginId)

/**
 * The connections a plugin lists in its rows, as the plugin of that id calls them, told of a change of
 * the rows by `subscribeChanges`.
 */
export const getPluginConnections = (
  pluginId: string,
  subscribeChanges: (handler: () => void) => () => void,
): ReturnType<typeof createPluginConnections> =>
  createPluginConnections(connectionCalls, pluginId, subscribeChanges)
