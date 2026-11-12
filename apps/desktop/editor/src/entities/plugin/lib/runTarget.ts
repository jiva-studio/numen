/**
 * The target a run of a command is over, in the words of the window and of the SDK. A run that names
 * a target is over that target alone, its file the path it names; a run that names none, as from a
 * bar, is over the tab in front.
 */
import type { CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'

/** The target of the tab in front, replaced by what the run names, if it names a target. */
export function getNamedTarget(front: CommandTarget, target: RunTarget | undefined): CommandTarget {
  if (!target) return { ...front, isTargetNamed: true }
  return {
    ...front,
    path: target.path,
    file: target.path,
    title: target.title,
    isFolder: target.isFolder,
    others: target.others,
    tab: target.tab || front.tab,
    isTargetNamed: true,
  }
}

/** The target in the words of the SDK, as the items of a command and a run with an item take it. */
export const toRunTarget = (target: CommandTarget): RunTarget => ({
  path: target.path,
  title: target.title,
  isFolder: target.isFolder ?? false,
  others: target.others ?? [],
  tab: target.tab,
})
