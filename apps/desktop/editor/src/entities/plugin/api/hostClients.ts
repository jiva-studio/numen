/**
 * The calls of the core that the plugin host makes for plugins: the agent they ask and the texts
 * they read.
 */
import { createAgentPort, createPluginTexts } from '@numen/plugin-loader'
import { agentService, texts } from '@/shared/api/clients'

/** The port through which the agent is asked and a conversation with it ends. */
export const agentPort = createAgentPort(agentService)

/** The texts of the vault, as the plugin of that id reads them. */
export const getPluginTexts = (pluginId: string): ReturnType<typeof createPluginTexts> =>
  createPluginTexts(texts, pluginId)
