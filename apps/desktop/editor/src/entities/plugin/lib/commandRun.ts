import type { WhenReader } from '@numen/plugin-loader'

function isOverTarget(command: { readonly when?: string }, whenReader: WhenReader): boolean {
  return (
    whenReader.getCondition(command.when)?.names.some((name) => name.startsWith('target.')) ?? false
  )
}

/** The path a run of the command is sent over: the file where the command reads its target, offers items, or a place named it; otherwise none. */
export function getRunPath(
  command: { readonly when?: string; readonly hasItems?: boolean },
  run: { readonly path: string; readonly isTargetNamed?: boolean },
  whenReader: WhenReader,
): string {
  return isOverTarget(command, whenReader) || command.hasItems || run.isTargetNamed ? run.path : ''
}
