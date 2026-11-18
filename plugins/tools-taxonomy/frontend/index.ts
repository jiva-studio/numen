/**
 * Frontend entrypoint for the Taxonomy plugin: the tags of a file as dots at the end of its row of the
 * file tree.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createTaxonomyApi } from './api/createTaxonomyApi'
import { createFileTags } from './file-tags'
import { createTagMap } from './model/createTagMap'

/** The id the manifest places in the slot fileTree.rowEnd. */
const FILE_TAGS_VIEW = 'file-tags'

export const activate = (plugin: PluginContext): void => {
  const tagMap = createTagMap({ api: createTaxonomyApi() })
  plugin.register.slotView({ id: FILE_TAGS_VIEW, component: createFileTags(tagMap) })
  plugin.identity.subscriptions.push({ dispose: () => tagMap.stop() })
}

export default definePlugin({
  activate,
})
