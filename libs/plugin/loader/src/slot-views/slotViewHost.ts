/**
 * The view a plugin registered for a slot of the window itself (`titleBar`, `statusBar`), drawn with
 * the stylesheet of its plugin. Such a view is drawn in the window, not in a pane of its own, so the
 * host adds the plugin's stylesheet while the view is drawn and removes it with the view.
 * Without a view it draws nothing.
 */
import { computed, defineComponent, h, onBeforeUnmount, watch, type PropType } from 'vue'
import { adoptPluginStyle, getPluginStyle } from '../styles'
import type { DeclaredSlotView } from './slotViews'

function addPluginStyle(pluginId: string): () => void {
  const release = adoptPluginStyle(pluginId, document)
  if (release) return release
  const href = getPluginStyle(pluginId)
  if (!href) return () => undefined
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = href
  link.dataset['pluginId'] = pluginId
  document.head.append(link)
  return () => link.remove()
}

/** Keeps the stylesheet of each plugin whose view is drawn, and releases it once none is. */
export function usePluginStyles(getViews: () => readonly { readonly pluginId: string }[]): void {
  const releases = new Map<string, () => void>()

  const sync = (views: readonly { readonly pluginId: string }[]): void => {
    const ids = new Set(views.map((one) => one.pluginId))
    for (const [id, release] of releases) {
      if (ids.has(id)) continue
      release()
      releases.delete(id)
    }
    for (const id of ids) if (!releases.has(id)) releases.set(id, addPluginStyle(id))
  }

  watch(getViews, sync, { immediate: true })
  onBeforeUnmount(() => sync([]))
}

/** The view of a slot of the window, with the stylesheet of its plugin; nothing without a view. */
export const SlotViewHost = defineComponent({
  name: 'SlotViewHost',
  props: {
    view: { type: Object as PropType<DeclaredSlotView | null>, default: null },
  },
  setup(props) {
    const views = computed(() => (props.view ? [props.view] : []))
    usePluginStyles(() => views.value)
    return () =>
      props.view ? h(props.view.registration.component, { key: props.view.pluginId }) : null
  },
})
