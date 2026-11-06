/**
 * Frontend entrypoint for the Media Importer plugin: a note of type `link` opens in its tab, web
 * addresses dropped from outside the window become link notes, and the media a link publishes are
 * offered to download.
 */
import { definePlugin, getFileStem, type PluginContext } from '@numen/plugin-sdk'
import LinkTab from './LinkTab.vue'
import './styles.css'
import { createLinks, links as servedLinks, refreshLinks, setTaskHost } from './api/wire'
import { getUrls, URI_LIST } from './lib/uriList'
import type { Links, Outcome } from './types'
import { WORDS as words } from './words'

export { LinkTab }

/** The note type a link carries, and the kind its tab is filed under. */
export const LINK = 'link'

/** What a link's tab holds: the note of the link. */
export interface LinkTabState {
  readonly path: string
}

/** Makes link notes of the addresses in a folder, as the plugin's backend does. */
export type CreateLinks = (
  folder: string,
  urls: readonly string[],
) => Promise<Outcome<readonly string[]>>

/** The items of app:link.downloadMedia: the video of a link, or its sound alone. */
const MEDIA = ['video', 'audio'] as const

const isMedia = (id: string | undefined): id is (typeof MEDIA)[number] =>
  MEDIA.some((one) => one === id)

export const activate = (
  context: PluginContext,
  create: CreateLinks = createLinks,
  links: Links = servedLinks,
): void => {
  setTaskHost(context.tasks)
  context.onVaultChange?.((event) => refreshLinks(event.paths ?? []))
  context.registerDropHandler?.({
    types: [URI_LIST],
    drop: async (folder, data) => {
      const made = await create(folder, getUrls(data[URI_LIST] ?? ''))
      if (made.ok) return made.value
      context.workspace?.showError(made.error)
      return []
    },
  })
  context.registerCommand({
    id: 'app:link.downloadMedia',
    listItems: async (target) => {
      const facts = await links.getLink(target.path)
      return facts.ok && facts.value.isCopyable ? MEDIA.map((id) => ({ id, title: words[id] })) : []
    },
    handler: async (note, argument) => {
      if (!isMedia(argument)) return
      const outcome = await links.copyMedia(note.path, argument)
      if (!outcome.ok) context.workspace?.showError(outcome.error)
    },
  })
  const getTitle = (path: string) => getFileStem(path) || words.link
  context.registerTabKind<LinkTabState>({
    kind: LINK,
    component: LinkTab,
    open: (path) => ({ path }),
    getTitle: (state) => getTitle(state.path),
    identity: (path) => path,
    getTarget: (state) => ({ path: state.path, file: state.path, title: getTitle(state.path) }),
    getOpenTab: (state) => ({ path: state.path }),
  })
}

export default definePlugin({
  activate,
})
