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
  readonly vectors: number
  readonly chunks: number
  readonly waiting: number
  readonly sources: number
  readonly kinds: readonly KindCount[]
  readonly model: string
  readonly recipe: string
  readonly dimensions: number
  readonly profile: string
  readonly threads: number
  readonly speed?: number
  readonly changedAt?: string
  readonly databaseBytes: number
  readonly vectorBytes: number
}
