/**
 * Frontend entrypoint for the Media Importer plugin: a note of type `link` opens in its tab, and web
 * addresses dropped from outside the window become link notes.
 */
import { definePlugin, getFileStem, type PluginContext } from '@numen/plugin-sdk'
import LinkTab from './LinkTab.vue'
import './styles.css'
import { createLinks, setTaskHost } from './api/wire'
import { getUrls, URI_LIST } from './lib/uriList'
import type { Outcome } from './types'
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

export const activate = (context: PluginContext, create: CreateLinks = createLinks): void => {
  setTaskHost(context.tasks)
  context.registerDropHandler?.({
    types: [URI_LIST],
    drop: async (folder, data) => {
      const made = await create(folder, getUrls(data[URI_LIST] ?? ''))
      if (made.ok) return made.value
      context.workspace?.showError(made.error)
      return []
    },
  })
  const getTitle = (path: string) => getFileStem(path) || words.link
  context.registerTabKind<LinkTabState>({
    kind: LINK,
    component: LinkTab,
    open: (path) => ({ path }),
    getTitle: (state) => getTitle(state.path),
    identity: (path) => path,
    getTarget: (state) => ({ path: state.path, title: getTitle(state.path) }),
    getOpenTab: (state) => ({ path: state.path }),
  })
}

export default definePlugin({
  activate,
})
