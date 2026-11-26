import { defineCollection } from 'astro:content'
import { readRegistry } from './components/plugins/registry'

/**
 * The plugins of the registry, one entry per plugin id. The README is rendered as the entry's
 * content and each release note as `notesHtml`, with Astro's own Markdown processor;
 * `featuredRank` is the plugin's place in the registry's featured list, or null.
 */
const plugins = defineCollection({
  loader: {
    name: 'numen-plugin-registry',
    load: async ({ store, renderMarkdown, logger }) => {
      const registry = await readRegistry()
      store.clear()
      for (const plugin of registry.plugins) {
        const changelog = await Promise.all(
          plugin.changelog.map(async (change) => ({
            ...change,
            notesHtml: change.notes ? (await renderMarkdown(change.notes)).html : null,
          })),
        )
        const rank = registry.featured.indexOf(plugin.id)
        const data = { ...plugin, changelog, featuredRank: rank === -1 ? null : rank }
        if (plugin.readme) {
          store.set({ id: plugin.id, data, rendered: await renderMarkdown(plugin.readme.text) })
        } else {
          store.set({ id: plugin.id, data })
        }
      }
      logger.info(`${registry.plugins.length} plugins, registry ${registry.contentHash.slice(0, 12)}`)
    },
  },
})

export const collections = { plugins }
