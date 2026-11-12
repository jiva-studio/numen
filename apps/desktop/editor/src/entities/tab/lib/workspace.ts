/**
 * What the window opens with, and the names of its tabs.
 *
 * The workspace takes identities and returns them; which component each one maps to
 * is decided in `App.vue`.
 */
import { branch, pane, type Workspace } from '@numen/workbench'

/** What this installation is configured as, one tab of it to a window. */
export const SETTINGS = 'settings'

/**
 * The main tabs, and one pane along the trailing edge holding the aside tabs, the first
 * of each in front. The user starts in the main pane. Where one side is empty, the
 * other fills the window.
 */
export const createWorkspace = (main: readonly string[], aside: readonly string[]): Workspace => {
  const only = aside.length === 0 ? main : aside
  const root =
    main.length > 0 && aside.length > 0
      ? branch('root', [pane('main', main), pane('aside', aside)], [0.72, 0.28])
      : pane('main', only)
  return { root, axis: 'horizontal', focus: 'main' }
}
