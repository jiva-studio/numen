import { create } from '@bufbuild/protobuf'
import {
  CatalogEntrySchema,
  ManifestContributesSchema,
  ManifestSchema,
  ManifestTabKindSchema,
} from '@numen/protocol'
import {
  CATALOG_STATE,
  type CatalogEntry,
  type CatalogIncompatibilityValue,
  type CatalogStateValue,
  type Manifest,
  type PluginCatalog,
} from '../catalog/catalog'
import type { PluginInfo } from '../catalog/plugins'
import { asFailure, asValue } from '../result'

/** A manifest for a test: the fields given, and the schema's default for the rest. */
export function createMockManifest(
  id: string,
  fields: {
    name?: string
    version?: string
    permissions?: readonly string[]
    frontend?: boolean
  } = {},
): Manifest {
  return create(ManifestSchema, {
    id,
    name: fields.name ?? id,
    version: fields.version ?? '1.0.0',
    permissions: [...(fields.permissions ?? [])],
    ...(fields.frontend ? { frontend: { entrypoint: 'dist/index.js', styles: '' } } : {}),
  })
}

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

/** A plugin the host lists, with what a test cares about put over the defaults. */
export function createMockPluginInfo(
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
): PluginInfo {
  return {
    id,
    name,
    version: '1.0.0',
    description: '',
    isActive: false,
    settings: [],
    capabilities: [],
    secrets: [],
    storedSecrets: [],
    consent: 'none',
    hasFrontend: false,
    ...overrides,
  }
}
