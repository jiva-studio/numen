import { create } from '@bufbuild/protobuf'
import { CatalogEntrySchema, ManifestContributesSchema, ManifestTabKindSchema } from '@numen/protocol'
import { asFailure, asValue } from '@numen/plugin-sdk'
import {
  CATALOG_STATE,
  type CatalogEntry,
  type CatalogIncompatibilityValue,
  type CatalogStateValue,
  type PluginCatalog,
} from '../shared/pluginCatalog'

/** What a catalog entry made for a test is given. */
export interface MockCatalogEntryOptions {
  readonly name?: string
  readonly version?: string
  readonly description?: string
  readonly category?: string
  readonly feature?: string
  readonly state?: CatalogStateValue
  readonly incompatibility?: CatalogIncompatibilityValue
  readonly incompatibilityDetail?: string
  /** The kinds of tab the plugin provides. */
  readonly tabKinds?: readonly (string | { readonly kind: string; readonly title: string })[]
}

/** A catalog entry for the plugin `id`, installed unless the options say otherwise. */
export function createMockCatalogEntry(
  id: string,
  options: MockCatalogEntryOptions = {},
): CatalogEntry {
  return create(CatalogEntrySchema, {
    id,
    name: options.name ?? id,
    version: options.version ?? '1.0.0',
    description: options.description ?? '',
    category: options.category ?? '',
    source: 'mock',
    state: options.state ?? CATALOG_STATE.INSTALLED,
    ...(options.incompatibility === undefined
      ? {}
      : {
          incompatibility: options.incompatibility,
          incompatibilityDetail: options.incompatibilityDetail ?? '',
        }),
    contributes: create(ManifestContributesSchema, {
      feature: options.feature ?? '',
      category: options.category ?? '',
      tabKinds: (options.tabKinds ?? []).map((one) =>
        create(ManifestTabKindSchema, typeof one === 'string' ? { kind: one } : one),
      ),
    }),
  })
}

/** A PluginCatalog over a list of entries that a test changes through `setEntries`. */
export interface MockPluginCatalog extends PluginCatalog {
  setEntries(entries: readonly CatalogEntry[]): void
  /** The kinds of tab lookups for, in order. */
  getAskedKinds(): readonly string[]
}

export function createMockPluginCatalog(initial: readonly CatalogEntry[] = []): MockPluginCatalog {
  let entries = initial
  const lookups: string[] = []
  return {
    setEntries: (next) => {
      entries = next
    },
    getAskedKinds: () => lookups,
    list: () => Promise.resolve(asValue({ entries, sourceErrors: [] })),
    get: (id) => {
      const entry = entries.find((one) => one.id === id)
      return Promise.resolve(
        entry
          ? asValue({ entry, sourceErrors: [] })
          : asFailure({ kind: 'not-found', sourceErrors: [] }),
      )
    },
    findByTabKind: (kind) => {
      lookups.push(kind)
      const entry = entries.find((one) =>
        one.contributes?.tabKinds.some((tabKind) => tabKind.kind === kind),
      )
      return Promise.resolve(
        entry
          ? asValue({ entry, sourceErrors: [] })
          : asFailure({ kind: 'not-found', sourceErrors: [] }),
      )
    },
  }
}
