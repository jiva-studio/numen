import { createPluginModels } from '@numen/plugin-loader'
import { models } from '@/shared/api/clients'

/** The models of the connections a plugin lists in its rows, as the plugin of that id calls them. */
export const getPluginModels = (pluginId: string): ReturnType<typeof createPluginModels> =>
  createPluginModels(models, pluginId)
