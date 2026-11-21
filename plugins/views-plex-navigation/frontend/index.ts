/**
 * Frontend entrypoint for the Plex Navigation plugin: Back and Forward buttons in the slot at the
 * top of the board of the plex.
 */
import { defineComponent, h, onScopeDispose, ref, watch, type PropType } from 'vue'
import {
  definePlugin,
  type HostServices,
  type PlexBoardViewProps,
  type PluginContext,
} from '@numen/plugin-sdk'
import { createBoardHistories, type BoardHistories } from './model/history'
import PlexNavigation from './ui/PlexNavigation.vue'

/** The id the manifest declares the view under. */
const NAVIGATION_VIEW = 'navigation'

export function createNavigationView(histories: BoardHistories) {
  return defineComponent({
    name: 'PlexNavigationView',
    props: {
      board: { type: String, required: true },
      path: { type: String, required: true },
      go: { type: Function as PropType<PlexBoardViewProps['go']>, required: true },
    },
    setup: (props) => {
      const changes = ref(0)
      const move = (node: string | null) => {
        changes.value += 1
        if (node !== null) props.go(node)
      }
      watch(
        () => [props.board, props.path] as const,
        ([board, path]) => {
          if (path === '') return
          histories.visit(board, path)
          changes.value += 1
        },
        { immediate: true },
      )
      onScopeDispose(() => histories.forget(props.board))
      return () => {
        const isCurrent = changes.value >= 0
        return h(PlexNavigation, {
          canBack: isCurrent && histories.canBack(props.board),
          canForward: isCurrent && histories.canForward(props.board),
          onBack: () => move(histories.back(props.board)),
          onForward: () => move(histories.forward(props.board)),
        })
      }
    },
  })
}

export function activate(plugin: PluginContext, _host: HostServices): void {
  plugin.register.slotView({
    id: NAVIGATION_VIEW,
    component: createNavigationView(createBoardHistories()),
  })
}

export default definePlugin({
  activate,
})
