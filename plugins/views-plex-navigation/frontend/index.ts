/**
 * Frontend entrypoint for the Plex Navigation plugin: Back and Forward buttons in the slot at the
 * top of the board of the plex.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { createBoardHistories, type BoardHistories } from './model/history'
import type { BoardKey } from './types'
import PlexNavigation from './ui/PlexNavigation.vue'

/** The id the manifest declares the view under. */
const NAVIGATION_VIEW = 'navigation'

interface NavigationSource {
  readonly histories: BoardHistories
  readonly getBoard: () => BoardKey | null
}

export function createNavigationView({ histories, getBoard }: NavigationSource) {
  return defineComponent({
    name: 'PlexNavigation',
    setup: () => () => {
      const board = getBoard()
      return h(PlexNavigation, {
        canBack: board !== null && histories.canBack(board),
        canForward: board !== null && histories.canForward(board),
      })
    },
  })
}

export function activate(plugin: PluginContext, _host: HostServices): void {
  const source: NavigationSource = { histories: createBoardHistories(), getBoard: () => null }
  plugin.register.slotView({ id: NAVIGATION_VIEW, component: createNavigationView(source) })
}

export default definePlugin({
  activate,
})
