/**
 * The name a row is given, as the path the file is filed under from now on.
 */
import { getFolderPath, ROOT } from './useFileTree'
import { fileOf } from '@/shared/paths'

/**
 * Where a name's ending begins, and nowhere for a name carrying none.
 */
const findExtensionStart = (name: string): number => {
  const cut = name.lastIndexOf('.')
  return cut >= 0 && !/\s/u.test(name.slice(cut)) ? cut : -1
}

/**
 * The ending a name carries, the dot with it, and nothing where it carries none.
 */
const getExtension = (name: string): string => {
  const at = findExtensionStart(name)
  return at > 0 ? name.slice(at) : ''
}

/**
 * A name typed over a row, as the path the file is filed under from now on.
 */
export const resolveRenamePath = (path: string, name: string, folder = false): string => {
  const typed = name.trim()
  if (!typed || typed.includes('/')) return ''

  const carries = findExtensionStart(typed) >= 0
  const called = folder || carries ? typed : `${typed}${getExtension(fileOf(path))}`
  if (called === fileOf(path)) return ''

  const under = getFolderPath(path)
  return under === ROOT ? called : `${under}/${called}`
}
