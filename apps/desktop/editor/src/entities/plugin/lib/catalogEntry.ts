/**
 * One plugin as the catalog knows it, in the words the window uses. The category is the one the
 * plugin declares, else the one its source gives, and empty where neither does. `isAvailable` holds
 * where a source offers the plugin and it is not installed.
 */
export interface CatalogEntry {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  readonly category: string
  readonly feature: string
  readonly tabKinds: readonly string[]
  readonly isAvailable: boolean
}

/** A source of plugins that could not be read in full. */
interface CatalogSourceFailure {
  readonly source: string
  readonly message: string
}

/** The entries of every source and the sources that could not be read in full. */
export interface CatalogListing {
  readonly entries: readonly CatalogEntry[]
  readonly sourceFailures: readonly CatalogSourceFailure[]
}
