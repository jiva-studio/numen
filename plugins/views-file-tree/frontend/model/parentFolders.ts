/** The folders above a path, from the root down. */
export const getParentFolders = (path: string): readonly string[] => {
  const parts = path.split('/').slice(0, -1)
  return parts.map((_, deep) => parts.slice(0, deep + 1).join('/'))
}
