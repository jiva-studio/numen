/** The tags of the notes as the frontend holds them, and the messages of the tag map the backend sends. */

/** The tags of the notes by path; a note without tags has no entry. */
export type TagsByPath = Readonly<Record<string, readonly string[]>>

/**
 * One message of the stream WatchTagMap, as the Go message in backend/api encodes it: the first holds
 * the whole map in `tags`, each later one holds in `changed` the paths whose tags changed, with an
 * empty list for a path left with none.
 */
export interface TagMapMessage {
  readonly tags?: TagsByPath
  readonly changed?: TagsByPath
}
