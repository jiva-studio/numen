import { pluginService } from '@/shared/api/clients'
import type { Plugin } from '../lib/plugin'

/** The plugins the application has installed. */
export const listPlugins = async (): Promise<readonly Plugin[]> =>
  (await pluginService.getPlugins({})).plugins.map((one) => ({ id: one.id, name: one.name }))
