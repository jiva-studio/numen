import type { CommandTarget } from '@numen/plugin-sdk'

/** The folder a note from a template is made in: the target when it is a folder, the folder of the target when it is a file, the vault root when it has no path. */
export function getTargetFolder(target: Pick<CommandTarget, 'path' | 'isFolder'>): string {
  if (target.isFolder) return target.path
  return target.path.slice(0, Math.max(target.path.lastIndexOf('/'), 0))
}
