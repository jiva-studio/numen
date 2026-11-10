/**
 * The plugin that provides a kind of tab, as the catalog knows it.
 */
import {
  CATALOG_INCOMPATIBILITY,
  CATALOG_STATE,
  type CatalogEntry,
  type CatalogStateValue,
} from '@numen/core-client'
import { pluginCatalog } from '@/shared/api/clients'
import type { PluginStanding, TabKindPlugin } from '../lib/unavailableView'

const standings: Record<CatalogStateValue, PluginStanding> = {
  [CATALOG_STATE.UNSPECIFIED]: 'installed',
  [CATALOG_STATE.INSTALLED]: 'installed',
  [CATALOG_STATE.DISABLED]: 'disabled',
  [CATALOG_STATE.AVAILABLE]: 'offered',
}

const mapPlugin = (entry: CatalogEntry, kind: string): TabKindPlugin => ({
  id: entry.id,
  name: entry.name,
  feature: entry.contributes?.feature || entry.name,
  standing: standings[entry.state],
  incompatibility:
    entry.incompatibility === CATALOG_INCOMPATIBILITY.UNSPECIFIED
      ? null
      : entry.incompatibilityDetail,
  kindTitle: entry.contributes?.tabKinds.find((one) => one.kind === kind)?.title ?? '',
})

/** The plugin the catalog names for a kind of tab, or null where it names none or cannot be read. */
export const findTabKindPlugin = async (kind: string): Promise<TabKindPlugin | null> => {
  const answer = await pluginCatalog.findByTabKind(kind)
  return answer.ok ? mapPlugin(answer.value.entry, kind) : null
}
