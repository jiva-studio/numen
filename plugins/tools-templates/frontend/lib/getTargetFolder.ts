import type { CommandTarget } from '@numen/plugin-sdk'

/** The folder a note from a template is made in: the target when it is a folder, the folder of the target when it is a file, the vault root when it has no path. */
export function getTargetFolder(_target: Pick<CommandTarget, 'path' | 'isFolder'>): string {
  throw new Error('not implemented')
}
