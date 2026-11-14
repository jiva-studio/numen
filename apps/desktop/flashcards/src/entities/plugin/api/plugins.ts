import type { WindowClients } from '@/shared/api/clients'
import type { Plugin } from '../lib/plugin'

/** Lists through `pluginService` the plugins the application has installed. */
export const createPluginLister =
  (pluginService: WindowClients['pluginService']) => async (): Promise<readonly Plugin[]> =>
    (await pluginService.getPlugins({})).plugins.map((one) => ({ id: one.id, name: one.name }))

export type PluginLister = ReturnType<typeof createPluginLister>
