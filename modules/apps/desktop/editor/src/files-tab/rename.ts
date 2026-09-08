/**
 * The name a row is given, as the path the file is filed under from now on.
 *
 * A name is a name and not a path: the field renames, and dragging moves. What
 * a note calls itself follows where a title and a filename are kept as one
 * name, so a name carrying no ending keeps the one the file has.
 */
import { folderOf, ROOT } from './listing'
import { fileOf } from '../shared/paths'

/**
 * Where a name's ending begins, and nowhere for a name carrying none. An ending
 * is the last dot and what follows it, and what follows it holds no space.
 */
const endingAt = (name: string): number => {
  const cut = name.lastIndexOf('.')
  return cut >= 0 && !/\s/u.test(name.slice(cut)) ? cut : -1
}

/**
 * The ending a name carries, the dot with it, and nothing where it carries
 * none. A name that is a dot and an ending carries none: that is its whole
 * name, and it has none to lend.
 */
const endingOf = (name: string): string => {
  const at = endingAt(name)
  return at > 0 ? name.slice(at) : ''
}

/**
 * A name typed over a row, as the path the file is filed under from now on.
 * A name carrying no ending keeps the one the file has, so a note typed over
 * stays a note. A folder keeps whatever was typed.
 *
 * A name that is the one it carries, or that names a folder of its own, moves
 * nothing.
 */
export const renamedTo = (path: string, name: string, folder = false): string => {
  const typed = name.trim()
  if (!typed || typed.includes('/')) return ''

  const carries = endingAt(typed) >= 0
  const called = folder || carries ? typed : `${typed}${endingOf(fileOf(path))}`
  if (called === fileOf(path)) return ''

  const under = folderOf(path)
  return under === ROOT ? called : `${under}/${called}`
}
