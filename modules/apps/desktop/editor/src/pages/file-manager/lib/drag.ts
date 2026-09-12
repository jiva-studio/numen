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
 * The url a drag carries, and nothing where it carries none.
 *
 * The format is one url a line, and a line opening with `#` is a comment.
 * A drag of several is the first of them: one drop makes one file.
 */
export const getFirstUrl = (text: string): string =>
  text
    .split('\n')
    .map((line) => line.trim())
    .find((line) => line !== '' && !line.startsWith('#')) ?? ''

/** The url a drag let go over the tree carries, read out of what it holds. */
export const getDroppedUrl = (transfer: DataTransfer | null | undefined): string =>
  getFirstUrl(transfer?.getData(URI_LIST) ?? '')


