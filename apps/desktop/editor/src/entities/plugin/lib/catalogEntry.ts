/**
 * One plugin as the catalog knows it, in the words the window uses.
 */
export interface CatalogEntry {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  /** The category the plugin declares, else the one its source gives; empty where neither does. */
  readonly category: string
  readonly feature: string
  /** The kinds of tab the plugin provides. */
  readonly tabKinds: readonly string[]
  /** Whether a source offers the plugin and it is not installed. */
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
