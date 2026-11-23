/**
 * Frontend entrypoint for the Taxonomy plugin: the tags of a file as dots at the end of its row of the
 * file tree, as chips under its node in the plex, and the command that sets the tags of a note.
 */
import { createApi, definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { createTaxonomyApi, SERVICE } from './api/createTaxonomyApi'
import { createFileTags } from './file-tags'
import { createPlexTags } from './plex-tags'
import { createTagMap } from './model/createTagMap'
import { createSetTags } from './set-tags'
import { createDeleteTag } from './delete-tag'

/** The id the manifest places in the slot fileTree.rowEnd. */
const FILE_TAGS_VIEW = 'file-tags'

/** The id the manifest places in the slot plexGraph.nodeBelow. */
const PLEX_TAGS_VIEW = 'plex-tags'

export const activate = (plugin: PluginContext, host: HostServices): void => {
  const api = createApi(SERVICE)
  const tagMap = createTagMap({ api: createTaxonomyApi(api), events: host.events })
  plugin.register.slotView({ id: FILE_TAGS_VIEW, component: createFileTags(tagMap) })
  plugin.register.slotView({ id: PLEX_TAGS_VIEW, component: createPlexTags(tagMap) })
  plugin.identity.subscriptions.push({ dispose: () => tagMap.stop() })
  const deleteTag = createDeleteTag({ api, setTags: tagMap.setTags, beginWrite: tagMap.beginWrite })
  plugin.register.command({
    id: 'tags.delete',
    listItems: deleteTag.listItems,
    handler: deleteTag.handler,
  })
  const { picker } = host.ui
  if (picker) {
    const setTags = createSetTags({
      api,
      picker,
      getTags: tagMap.getTags,
      setTags: tagMap.setTags,
      beginWrite: tagMap.beginWrite,
    })
    plugin.register.command({
      id: 'tags.set',
      handler: async (note) => {
        await tagMap.waitForLoad()
        await setTags.handler(note)
      },
    })
  }
}

export default definePlugin({
  activate,
})
