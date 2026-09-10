/**
 * What a drag from outside the window carries, as the files tree reads it.
 *
 * A file dropped in is a path and reaches the window's own drag and drop. An
 * address dragged out of a browser is not a file: it arrives as `text/uri-list`
 * and is read here.
 */

/** The type a drag carrying an address is offered under. */
const URI_LIST = 'text/uri-list'

/** Whether a drag carries an address at all. */
export const carriesAddress = (types: readonly string[] | undefined): boolean =>
  types?.includes(URI_LIST) ?? false

/**
 * The address a drag carries, and nothing where it carries none.
 *
 * The format is one address a line, and a line opening with `#` is a comment.
 * A drag of several is the first of them: one drop makes one file.
 */
export const addressIn = (written: string): string =>
  written
    .split('\n')
    .map((line) => line.trim())
    .find((line) => line !== '' && !line.startsWith('#')) ?? ''

/** The address a drag let go over the tree carries, read out of what it holds. */
export const addressDropped = (held: DataTransfer | null | undefined): string =>
  addressIn(held?.getData(URI_LIST) ?? '')


