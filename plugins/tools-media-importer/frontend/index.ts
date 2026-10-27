/**
 * Frontend entrypoint for the Media Importer plugin: a note of type `link` opens in its tab.
 */
import { definePlugin, getFileStem, type PluginContext } from '@numen/plugin-sdk'
import LinkTab from './LinkTab.vue'
import './styles.css'
import { setTaskHost } from './api/wire'
import { WORDS as words } from './words'

export { LinkTab }

/** The note type a link carries, and the kind its tab is filed under. */
export const LINK = 'link'

/** What a link's tab holds: the note of the link. */
export interface LinkTabState {
  readonly path: string
}

export const activate = (context: PluginContext): void => {
  setTaskHost(context.tasks)
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
