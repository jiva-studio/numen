/**
 * The stylesheets of the plugins that have a view in the status bar.
 *
 * The views are drawn in the window itself, not in a pane of their own, so nothing else adds the
 * stylesheet of the plugin that registered one. It is added while the plugin has a view, and taken
 * out when the view goes.
 */
import { onBeforeUnmount, watch } from 'vue'
import { adoptPluginStyle, getPluginStyle, type DeclaredStatusView } from '@numen/plugin-host'

/** Adds the stylesheet of a plugin to the window: as an adopted sheet, or as a link where the sheet was not read. */
function createStyle(pluginId: string): () => void {
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

export function usePluginStyles(getViews: () => readonly DeclaredStatusView[]): void {
  const releases = new Map<string, () => void>()

  const sync = (views: readonly DeclaredStatusView[]): void => {
    const ids = new Set(views.map((one) => one.pluginId))
    for (const [id, release] of releases) {
      if (ids.has(id)) continue
      release()
      releases.delete(id)
    }
    for (const id of ids) if (!releases.has(id)) releases.set(id, createStyle(id))
  }

  watch(getViews, sync, { immediate: true })
  onBeforeUnmount(() => sync([]))
}
