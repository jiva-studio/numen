/** The tags of the notes as the frontend holds them, and the messages of the tag map the backend sends. */

/** The tags of the notes by path; a note without tags has no entry. */
export type TagsByPath = Readonly<Record<string, readonly string[]>>

/** The answer of GetTagMap: the tags of the notes of the vault by path. */
export interface TagMapResponse {
  readonly tags: TagsByPath
}

/** A tag of the vault with the number of notes that have it, as the Go message TagCount encodes it. */
interface TagCount {
  readonly tag: string
  readonly count: number
}

/** The answer of GetTags: every tag of the vault, and the tags most recently set that a note still has, the latest first. */
export interface TagList {
  readonly tags: readonly TagCount[]
  readonly recent: readonly string[]
}

/** The request of SetTags: the tags to toggle on the notes at the paths. */
export interface SetTagsRequest {
  readonly paths: readonly string[]
  readonly toggles: readonly string[]
}

/** The tags a note has after SetTags, or the error that left it as it was. */
interface NoteTags {
  readonly path: string
  readonly tags: readonly string[]
  readonly error?: string
}

/** The answer of SetTags: one entry for each path, in the order of the request. */
export interface SetTagsResult {
  readonly notes: readonly NoteTags[]
}
