/**
 * What a drag from outside the window carries, as the files tree reads it.
 *
 * A file dropped in is a path and reaches the window's own drag and drop. A url
 * dragged out of a browser is not a file: it arrives as `text/uri-list` and is
 * read here.
 */

/** The type a drag carrying a url is offered under. */
const URI_LIST = 'text/uri-list'

/** Whether a drag carries a url at all. */
export const hasUrl = (types: readonly string[] | undefined): boolean =>
  types?.includes(URI_LIST) ?? false

/**
 * The urls text carries, in the order they stand.
 *
 * The format is one url a line, and a line opening with `#` is a comment.
 */
export const getUrls = (text: string): readonly string[] =>
  text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== '' && !line.startsWith('#'))

/** The first url text carries, or empty string. */
export const getFirstUrl = (text: string): string => getUrls(text)[0] ?? ''

/** The urls a drag let go over the tree carries, read out of what it holds. */
export const getDroppedUrls = (transfer: DataTransfer | null | undefined): readonly string[] =>
  getUrls(transfer?.getData(URI_LIST) ?? '')

/** The first url a drag let go over the tree carries. */
export const getDroppedUrl = (transfer: DataTransfer | null | undefined): string =>
  getFirstUrl(transfer?.getData(URI_LIST) ?? '')
