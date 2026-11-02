/**
 * What the frontend reads of the backend: the messages of the index stats service. Keys and meaning
 * match the Go messages in backend/api.
 */

/** The sort of source the file type of a source tells. */
export type Kind = 'notes' | 'pdf' | 'epub' | 'media' | 'other'

export type GetStatsRequest = Record<string, never>

/** The sources of one kind and the chunks that can carry a vector among those they hold. */
export interface KindCount {
  readonly kind: Kind
  readonly sources: number
  readonly chunks: number
}

/** What the index holds, and the model, profile and speed that fill it. */
export interface Stats {
  /** How many chunks carry a vector, how many can, and how many do not yet. */
  readonly vectors: number
  readonly chunks: number
  readonly waiting: number
  /** The notes and files in the index, and the split by kind. */
  readonly sources: number
  readonly kinds: readonly KindCount[]
  readonly model: string
  readonly recipe: string
  readonly dimensions: number
  /** The performance profile in force and the threads it grants a plugin. */
  readonly profile: string
  readonly threads: number
  /** Chunks embedded each second over the last ten seconds; present while chunks wait. */
  readonly speed?: number
  /** When the counts last changed, in RFC 3339; absent before the first change seen. */
  readonly changedAt?: string
  /** The size on disk of the index database and of the tables that hold vectors. */
  readonly databaseBytes: number
  readonly vectorBytes: number
}
