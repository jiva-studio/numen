import { shallowRef, watch, type Ref, type ShallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginInfo } from '@numen/plugin-manager'
import type { CatalogEntry, PluginCatalog } from '@numen/core-client'

/**
 * The entries of the plugin catalog, read again each time the plugins the host lists change: a
 * plugin enabled, installed or removed changes both lists together.
 */
export function useCatalogEntries(
  catalog: PluginCatalog,
  plugins: Readonly<Ref<readonly PluginInfo[]>>,
  reportError: (error: unknown) => void,
): Readonly<ShallowRef<readonly CatalogEntry[]>> {
  const entries = shallowRef<readonly CatalogEntry[]>([])

  const refresh = async (): Promise<void> => {
    const answer = await catalog.list()
    if (!answer.ok) {
      const reason = answer.error.kind === 'unavailable' ? answer.error.message : 'it lists nothing'
      reportError(new Error(`The plugin catalog is unavailable: ${reason}`))
      return
    }
    entries.value = answer.value.entries
    for (const failure of answer.value.sourceErrors) {
      reportError(new Error(`The plugin source ${failure.source} failed: ${failure.message}`))
    }
  }

  // A read that lists the same plugins in the same states changes nothing in the catalog.
  watch(
    () => plugins.value.map((one) => `${one.id}:${one.isActive}`).join('\n'),
    () => runInBackground(refresh(), reportError),
  )
  return entries
}
