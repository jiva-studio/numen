/**
 * The calls of the core that the plugin host makes for plugins: the agent they ask and the texts
 * they read, and the models their rows of connections call.
 */
import { createAgentPort, createPluginModels, createPluginTexts } from '@numen/plugin-loader'
import { agentService, models, texts } from '@/shared/api/clients'

/** The port through which the agent is asked and a conversation with it ends. */
export const agentPort = createAgentPort(agentService)

/** The texts of the vault, as the plugin of that id reads them. */
export const getPluginTexts = (pluginId: string): ReturnType<typeof createPluginTexts> =>
  createPluginTexts(texts, pluginId)

/** The models of the connections a plugin lists in its rows, as the plugin of that id calls them. */
export const getPluginModels = (pluginId: string): ReturnType<typeof createPluginModels> =>
  createPluginModels(models, pluginId)
