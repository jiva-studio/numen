import { ROOT, getFileName, getFolderOf } from './entries'

const findExtensionStart = (name: string): number => {
  const dot = name.lastIndexOf('.')
  return !/\s/u.test(name.slice(dot)) ? dot : -1
}

const getExtension = (name: string): string => {
  const start = findExtensionStart(name)
  return start > 0 ? name.slice(start) : ''
}

/** A name typed over a row, as the path the file is filed under from now on; empty where the name changes nothing. */
export function getRenamedPath(path: string, name: string, isFolder: boolean): string {
  const clean = name.trim()
  if (!clean || clean.includes('/')) return ''
  const hasExtension = findExtensionStart(clean) >= 0
  const target = isFolder || hasExtension ? clean : `${clean}${getExtension(getFileName(path))}`
  if (target === getFileName(path)) return ''
  const folder = getFolderOf(path)
  return folder === ROOT ? target : `${folder}/${target}`
}

/** Where a dropped row lands: in the folder it went into, or in the folder of the row it came before. */
export function getDropFolder(target: { into: string | null } | { before: string }): string {
  return 'into' in target ? (target.into ?? ROOT) : getFolderOf(target.before)
}
