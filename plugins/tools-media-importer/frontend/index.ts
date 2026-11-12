/**
 * Frontend entrypoint for the Media Importer plugin: web addresses dropped from outside the window
 * become link notes, and the media a link publishes are offered to download.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createLinks, links as servedLinks } from './api/wire'
import { getUrls, URI_LIST } from './lib/uriList'
import type { DroppedLinks, Links, Outcome } from './types'
import { WORDS as words } from './words'

/** Makes link notes of the addresses in a folder, as the plugin's backend does. */
export type CreateLinks = (
  folder: string,
  urls: readonly string[],
) => Promise<Outcome<DroppedLinks>>

/** The items of app:link.downloadMedia: the video of a link, or its sound alone. */
const MEDIA = ['video', 'audio'] as const

const isMedia = (id: string | undefined): id is (typeof MEDIA)[number] =>
  MEDIA.some((one) => one === id)

export const activate = (
  context: PluginContext,
  create: CreateLinks = createLinks,
  links: Links = servedLinks,
): void => {
  context.register.dropHandler?.({
    types: [URI_LIST],
    drop: async (folder, data) => {
      const made = await create(folder, getUrls(data[URI_LIST] ?? ''))
      if (!made.ok) {
        context.ui.workspace?.messages.showError(made.error)
        return []
      }
      if (made.value.failed.length > 0) {
        context.ui.workspace?.messages.showError(
          `${words.notCreated} ${made.value.failed.join(', ')}`,
        )
      }
      return made.value.paths
    },
  })
  context.register.command({
    id: 'app:link.downloadMedia',
    listItems: async (target) => {
      const facts = await links.getLink(target.path)
      return facts.ok && facts.value.isCopyable ? MEDIA.map((id) => ({ id, title: words[id] })) : []
    },
    handler: async (note, itemId) => {
      if (!isMedia(itemId)) return
      const outcome = await links.copyMedia(note.path, itemId)
      if (!outcome.ok) context.ui.workspace?.messages.showError(outcome.error)
    },
  })
}

export default definePlugin({
  activate,
})
