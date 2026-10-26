/**
 * Frontend entrypoint for the Media Importer plugin: a link opens in its own tab, and the window
 * makes a link of an address through it.
 */
import { definePlugin, getFileStem, type PluginContext } from '@numen/plugin-sdk'
import LinkTab from './LinkTab.vue'
import { links } from './api/wire'
import type { Links } from './types'
import { WORDS as words } from './words'

export { LinkTab }

/** The kind a link's tab is filed under, and the kind of file the window makes of an address. */
export const LINK = 'url'

/** What a link's tab holds: the link. */
export interface LinkTabState {
  readonly path: string
}

export function createActivate(served: Links) {
  return (context: PluginContext): void => {
    const getTitle = (path: string) => getFileStem(path) || words.link
    context.registerReader({
      id: 'link',
      fileExtensions: ['.url'],
      component: LinkTab,
      title: words.link,
      source: 'url',
    })
    context.registerTabKind<LinkTabState>({
      kind: LINK,
      component: LinkTab,
      source: 'url',
      open: (path) => ({ path }),
      getTitle: (state) => getTitle(state.path),
      identity: (path) => path,
      getTarget: (state) => ({ path: state.path, title: getTitle(state.path) }),
      getOpenTab: (state) => ({ path: state.path }),
      createFile: (folder, address) => served.createLink(folder, address),
    })
  }
}

export const activate = createActivate(links)

export default definePlugin({
  activate,
})
