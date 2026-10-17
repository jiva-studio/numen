import { defineComponent, h } from 'vue'
import type { AnyTabKind } from './kinds'

/** What a tab of an unavailable kind holds: the kind it was made for, and what it stood on. */
export interface UnavailableState {
  readonly kind: string
  readonly at: string
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
        state.at ? h('p', { class: 'text-xs' }, state.at) : null,
      ])
    }
  },
})

/**
 * The kind of a tab whose own kind is not registered. It keeps the missing kind's name and
 * what the tab stood on, so a saved session lists the tab as it was and the plugin takes it
 * back once it is available.
 */
export function createUnavailableKind(kind: string): AnyTabKind {
  return {
    kind,
    open: (at: string): UnavailableState => ({ kind, at }),
    getTitle: () => 'Plugin unavailable',
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).at }),
    pane: UnavailablePane,
    identity: (at: string) => at,
  }
}
