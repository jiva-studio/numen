import { defineComponent, h } from 'vue'
import type { AnyTabKind } from './kinds'

/** What a tab of an unavailable kind holds: the kind it was made for, and what it stood on. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
}

const UnavailablePane = defineComponent({
  name: 'UnavailableTab',
  props: { state: { type: Object, required: true } },
  setup(props) {
    return () => {
      const state = props.state as UnavailableState
      return h('div', { class: 'tab-unavailable p-4', role: 'status' }, [
        h('h3', { class: 'font-bold' }, 'Plugin unavailable'),
        h('p', `The plugin for "${state.kind}" is missing or disabled.`),
        state.targetPath ? h('p', { class: 'text-xs' }, state.targetPath) : null,
      ])
    }
  },
})

/**
 * The kind of a tab whose own kind is not registered. It keeps the missing kind's name and
 * what the tab stood on, so a saved session lists the tab as it was and the plugin takes it
 * back once it is available.
 */
export const createUnavailableKind = (kind: string): AnyTabKind => {
  return {
    kind,
    open: (targetPath: string): UnavailableState => ({ kind, targetPath }),
    getTitle: () => 'Plugin unavailable',
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
}
