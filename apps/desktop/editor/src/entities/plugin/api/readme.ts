import type { WindowClients } from '@/shared/api/clients'

/**
 * The README.md of an installed plugin over `pluginCatalog`, and an empty text where the plugin has
 * none or the core cannot answer.
 */
export const createReadmeReader =
  (pluginCatalog: WindowClients['pluginCatalog']) =>
  async (pluginId: string): Promise<string> => {
    const answer = await pluginCatalog.getReadme(pluginId)
    return answer.ok ? answer.value : ''
  }

export type ReadmeReader = ReturnType<typeof createReadmeReader>
