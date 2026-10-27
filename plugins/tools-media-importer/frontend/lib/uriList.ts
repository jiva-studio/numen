/**
 * The addresses a `text/uri-list` holds (RFC 2483): one a line, and a line opening with `#` a
 * comment.
 */

/** The type a drag of addresses out of a browser carries them under. */
export const URI_LIST = 'text/uri-list'

/** The addresses the list holds, in the order they stand. */
export const getUrls = (list: string): readonly string[] =>
  list
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== '' && !line.startsWith('#'))
