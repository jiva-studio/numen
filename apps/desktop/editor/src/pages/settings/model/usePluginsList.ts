/**
 * The plugin list of the settings tab: the plugins with their catalog entries, which of them are
 * enabled, and the search query kept in step with the one the tab remembers.
 */
import { computed, ref, watch } from 'vue'
import type { CatalogEntry } from '@/entities/plugin'
import { write, type PluginInfo } from '@/entities/settings'
import { createManagedFromEntries } from '../lib/pluginListing'
import type { InstallationSettings } from '../types'

export interface PluginsListDeps {
  readonly getSettings: () => Pick<InstallationSettings, 'getSetting' | 'write'>
  readonly getCatalog: () => readonly CatalogEntry[]
  readonly getQuery: () => string
  readonly getPlugins: () => readonly PluginInfo[]
  readonly updateQuery: (query: string) => void
}

export function usePluginsList(deps: PluginsListDeps) {
  const searchQuery = ref(deps.getQuery())

  const managed = computed(() => createManagedFromEntries(deps.getPlugins(), deps.getCatalog()))

  function isPluginEnabled(pluginId: string): boolean {
    const val = deps.getSettings().getSetting([pluginId, 'enabled'])
    return typeof val === 'boolean' ? val : true
  }

  const enabled = computed(() =>
    Object.fromEntries(deps.getPlugins().map((one) => [one.id, isPluginEnabled(one.id)])),
  )

  watch(deps.getQuery, (newQuery) => {
    searchQuery.value = newQuery
  })

  watch(searchQuery, (newQuery) => {
    if (deps.getQuery() !== newQuery) deps.updateQuery(newQuery)
  })

  function setEnabled(pluginId: string, isEnabled: boolean): void {
    deps.getSettings().write([{ at: [pluginId, 'enabled'], value: write(isEnabled) }])
  }

  return { searchQuery, managed, enabled, setEnabled }
}
