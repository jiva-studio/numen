import type { TabId, Workspace } from './node'
import type { Rect } from './rect'
import { getRect } from './drop'
import { paneWithTab } from './tree'

/** The box of the drawn pane that holds the tab, in viewport coordinates; null where none does. */
export function getTabPaneRect(workspace: Workspace, tab: TabId): Rect | null {
  const holder = paneWithTab(workspace.root, tab)
  if (!holder) return null
  const element = document.querySelector(`[data-workspace-pane="${CSS.escape(holder.id)}"]`)
  return element ? getRect(element) : null
}
