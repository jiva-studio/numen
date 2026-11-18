/**
 * Frontend entrypoint for the Taxonomy plugin: the tags of a file as dots at the end of its row of the
 * file tree and as chips under its node in the plex.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createTaxonomyApi } from './api/createTaxonomyApi'
import { createFileTags } from './file-tags'
import { createPlexTags } from './plex-tags'
import { createTagMap } from './model/createTagMap'

/** The id the manifest places in the slot fileTree.rowEnd. */
const FILE_TAGS_VIEW = 'file-tags'
/** The id the manifest places in the slot plexGraph.nodeBelow. */
const PLEX_TAGS_VIEW = 'plex-tags'

export const activate = (plugin: PluginContext): void => {
  const tagMap = createTagMap({ api: createTaxonomyApi() })
  plugin.register.slotView({ id: FILE_TAGS_VIEW, component: createFileTags(tagMap) })
  plugin.register.slotView({ id: PLEX_TAGS_VIEW, component: createPlexTags(tagMap) })
  plugin.identity.subscriptions.push({ dispose: () => tagMap.stop() })
}

export default definePlugin({
  activate,
})
