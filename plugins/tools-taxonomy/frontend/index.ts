/**
 * Frontend entrypoint for the Taxonomy plugin: the tags of a file as dots at the end of its row of the
 * file tree, and the command that sets the tags of a note.
 */
import { createApi, definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { createTaxonomyApi, SERVICE } from './api/createTaxonomyApi'
import { createFileTags } from './file-tags'
import { createTagMap } from './model/createTagMap'
import { createSetTags } from './set-tags'

/** The id the manifest places in the slot fileTree.rowEnd. */
const FILE_TAGS_VIEW = 'file-tags'

export const activate = (plugin: PluginContext, host: HostServices): void => {
  const api = createApi(SERVICE)
  const tagMap = createTagMap({ api: createTaxonomyApi(api) })
  plugin.register.slotView({ id: FILE_TAGS_VIEW, component: createFileTags(tagMap) })
  plugin.identity.subscriptions.push({ dispose: () => tagMap.stop() })
  const { picker } = host.ui
  if (picker) plugin.register.command(createSetTags({ api, picker, getTags: tagMap.getTags }))
}

export default definePlugin({
  activate,
})
