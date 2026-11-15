/**
 * Frontend entrypoint of the saved search plugin: the tab of a note of type search, which shows the
 * query of the note and the results its last run cached, and opens a result at its place.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { createSavedSearchKind, createSavedSearchService, SavedSearchTab } from './results'
import './styles.css'

export const activate = (plugin: PluginContext, host: HostServices): void => {
  const workspace = host.ui.workspace
  if (!workspace) return
  plugin.register.tabKind(
    createSavedSearchKind({
      service: createSavedSearchService(),
      openAt: (path, ranges) => workspace.files.openAt(path, ranges),
      locale: navigator.language,
      component: SavedSearchTab,
    }),
  )
}

export default definePlugin({
  activate,
})
