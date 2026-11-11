import { runInBackground } from '@numen/ui'
import type { MovedTab } from '@/entities/tab'
import { createMovedTabOpener, type MovedTabOpenerDeps } from './tabMove'

/** Opens a tab sent from another window without waiting; the opener reports its own failures. */
export function createMovedTabStarter(deps: MovedTabOpenerDeps): (moved: MovedTab) => void {
  const open = createMovedTabOpener(deps)
  return (moved) => runInBackground(open(moved), deps.onError)
}
