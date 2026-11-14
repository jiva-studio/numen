import { shallowRef, watch, type Ref, type ShallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginInfo } from '@numen/plugin-manager'
import { ReportedError, type Caught } from '@/shared/api/errors'
import type { CatalogReader } from '../api/catalog'
import type { CatalogEntry } from '../lib/catalogEntry'

/**
 * The entries of the plugin catalog, read through `listCatalog` again each time the plugins the
 * host lists change: a plugin enabled, installed or removed changes both lists together.
 */
export function useCatalogEntries(
  listCatalog: CatalogReader,
  plugins: Readonly<Ref<readonly PluginInfo[]>>,
  reportError: (error: Caught) => void,
): Readonly<ShallowRef<readonly CatalogEntry[]>> {
  const entries = shallowRef<readonly CatalogEntry[]>([])

  const refresh = async (): Promise<void> => {
    const answer = await listCatalog()
    if (!answer.ok) {
      reportError(new ReportedError(`The plugin catalog is unavailable: ${answer.error}`))
      return
    }
    entries.value = answer.value.entries
    for (const failure of answer.value.sourceFailures) {
      reportError(
        new ReportedError(`The plugin source ${failure.source} failed: ${failure.message}`),
      )
    }
  }

  // A read that lists the same plugins in the same states changes nothing in the catalog.
  watch(
    () => plugins.value.map((one) => `${one.id}:${one.isActive}`).join('\n'),
    () => runInBackground(refresh(), reportError),
  )
  return entries
}
