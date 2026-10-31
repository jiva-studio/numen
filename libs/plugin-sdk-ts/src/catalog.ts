/**
 * The plugin catalog: the plugins the application has, the ones it could have, and the plugin that
 * provides a kind of tab. Entries and manifests are the generated protocol types.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import { fromJson, type JsonValue } from '@bufbuild/protobuf'
import {
  CatalogErrorCode,
  CatalogIncompatibility,
  CatalogService,
  CatalogState,
  ManifestSchema,
  type CatalogEntry as CatalogEntryMessage,
  type CatalogSourceError as CatalogSourceErrorMessage,
  type Manifest as ManifestMessage,
  type ManifestContributes as ManifestContributesMessage,
  type ManifestTabKind as ManifestTabKindMessage,
} from '@numen/protocol'
import { asFailure, asValue, type Result } from './result'
import { formatErrorMessage } from './errors/error'
import { transport } from './transport'

/** One plugin as the catalog knows it. */
export type CatalogEntry = CatalogEntryMessage
/** A source that could not be read in full. */
export type CatalogSourceError = CatalogSourceErrorMessage
/** The schema of a plugin's manifest.json. */
export type Manifest = ManifestMessage
/** What a plugin adds to the application, known before its frontend is loaded. */
export type ManifestContributes = ManifestContributesMessage
/** A kind of tab a plugin registers. */
export type ManifestTabKind = ManifestTabKindMessage

export const createCatalogClient = (customTransport: Transport = transport) =>
  createClient(CatalogService, customTransport)

export const catalogService = createCatalogClient()

export const CATALOG_STATE = CatalogState
export type CatalogStateValue = CatalogState

export const CATALOG_INCOMPATIBILITY = CatalogIncompatibility
export type CatalogIncompatibilityValue = CatalogIncompatibility

export const CATALOG_ERROR_CODE = CatalogErrorCode
export type CatalogErrorCodeValue = CatalogErrorCode

/** The calls of CatalogService the catalog is read through. */
export type CatalogClient = Pick<
  Client<typeof CatalogService>,
  'listCatalog' | 'getCatalogEntry' | 'findCatalogEntryByTabKind'
>

/** The entries of every source and the sources that could not be read in full. */
export interface CatalogListing {
  readonly entries: readonly CatalogEntry[]
  readonly sourceErrors: readonly CatalogSourceError[]
}

/** An entry that was found, and the sources that failed while it was looked for. */
export interface CatalogMatch {
  readonly entry: CatalogEntry
  readonly sourceErrors: readonly CatalogSourceError[]
}

/** Why no entry was returned: the catalog has none, or the call failed. */
export type CatalogFailure =
  | { readonly kind: 'not-found'; readonly sourceErrors: readonly CatalogSourceError[] }
  | { readonly kind: 'unavailable'; readonly message: string }

/** The plugin catalog of the application. */
export interface PluginCatalog {
  list(): Promise<Result<CatalogListing, CatalogFailure>>
  get(id: string): Promise<Result<CatalogMatch, CatalogFailure>>
  findByTabKind(kind: string): Promise<Result<CatalogMatch, CatalogFailure>>
}

interface EntryAnswer {
  readonly error: CatalogErrorCode
  readonly entry?: CatalogEntry | undefined
  readonly sourceErrors: CatalogSourceError[]
}

const readMatch = (answer: EntryAnswer): Result<CatalogMatch, CatalogFailure> => {
  if (answer.error === CatalogErrorCode.NOT_FOUND || answer.entry === undefined) {
    return asFailure({ kind: 'not-found', sourceErrors: answer.sourceErrors })
  }
  return asValue({ entry: answer.entry, sourceErrors: answer.sourceErrors })
}

const readUnavailable = (error: unknown): Result<never, CatalogFailure> =>
  asFailure({ kind: 'unavailable', message: formatErrorMessage(error) })

/** The catalog read through a CatalogService client. */
export const createPluginCatalog = (client: CatalogClient): PluginCatalog => ({
  list: async () => {
    try {
      const answer = await client.listCatalog({})
      return asValue({ entries: answer.entries, sourceErrors: answer.sourceErrors })
    } catch (error) {
      return readUnavailable(error)
    }
  },
  get: async (id) => {
    try {
      return readMatch(await client.getCatalogEntry({ id }))
    } catch (error) {
      return readUnavailable(error)
    }
  },
  findByTabKind: async (kind) => {
    try {
      return readMatch(await client.findCatalogEntryByTabKind({ kind }))
    } catch (error) {
      return readUnavailable(error)
    }
  },
})

/** The catalog of the window the page is served from. */
export const pluginCatalog = createPluginCatalog(catalogService)

/**
 * The text of a manifest.json read against the generated schema. Keys the schema does not name are
 * ignored; a value of the wrong type is an error.
 */
export const parseManifest = (text: string): Result<Manifest, string> => {
  try {
    const document = JSON.parse(text) as JsonValue
    return asValue(fromJson(ManifestSchema, document, { ignoreUnknownFields: true }))
  } catch (error) {
    return asFailure(formatErrorMessage(error))
  }
}
