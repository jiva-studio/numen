import { createPluginConnections } from '@numen/plugin-loader'
import { connectionCalls } from '@/shared/api/clients'

/** The connections a plugin lists in its rows, as the plugin of that id calls them. */
export const getPluginConnections = (
  pluginId: string,
): ReturnType<typeof createPluginConnections> => createPluginConnections(connectionCalls, pluginId)
