/**
 * The plugin catalog: the plugins the application has and the ones it could have.
 */
import { CATALOG_STATE } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { Result } from '@/shared/lib/result'
import type { CatalogListing } from '../lib/catalogEntry'

/**
 * Every entry of the catalog read through `pluginCatalog`, and the sources that failed. A catalog
 * that cannot be read answers why, and a catalog that lists nothing says so.
 */
export const createCatalogReader =
  (pluginCatalog: WindowClients['pluginCatalog']) => async (): Promise<Result<CatalogListing>> => {
    const answer = await pluginCatalog.list()
    if (!answer.ok) {
      return {
        ok: false,
        error: answer.error.kind === 'unavailable' ? answer.error.message : 'it lists nothing',
      }
    }
    return {
      ok: true,
      value: {
        entries: answer.value.entries.map((entry) => ({
          id: entry.id,
          name: entry.name,
          version: entry.version,
          description: entry.description,
          category: entry.contributes?.category || entry.category || '',
          feature: entry.contributes?.feature ?? '',
          tabKinds: entry.contributes?.tabKinds.map((one) => one.kind) ?? [],
          isAvailable: entry.state === CATALOG_STATE.AVAILABLE,
        })),
        sourceFailures: answer.value.sourceErrors.map((one) => ({
          source: one.source,
          message: one.message,
        })),
      },
    }
  }

export type CatalogReader = ReturnType<typeof createCatalogReader>
