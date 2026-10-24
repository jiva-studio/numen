/**
 * The name a row is given, as the path the file is filed under from now on.
 */
import { getFolderPath, ROOT } from './useFileTree'
import { getFileName } from '@numen/plugin-sdk'

/**
 * Where a name's ending begins, and nowhere for a name carrying none.
 */
const findExtensionStart = (name: string): number => {
  const dotIndex = name.lastIndexOf('.')
  return !/\s/u.test(name.slice(dotIndex)) ? dotIndex : -1
}

/**
 * The ending a name carries, the dot with it, and nothing where it carries none.
 */
const getExtension = (name: string): string => {
  const extStart = findExtensionStart(name)
  return extStart > 0 ? name.slice(extStart) : ''
}

/**
 * A name typed over a row, as the path the file is filed under from now on.
 */
export const resolveRenamePath = (path: string, name: string, isFolder = false): string => {
  const cleanName = name.trim()
  if (!cleanName || cleanName.includes('/')) return ''

  const hasExtension = findExtensionStart(cleanName) >= 0
  const targetName =
    isFolder || hasExtension ? cleanName : `${cleanName}${getExtension(getFileName(path))}`
  if (targetName === getFileName(path)) return ''

  const folderPath = getFolderPath(path)
  return folderPath === ROOT ? targetName : `${folderPath}/${targetName}`
}
