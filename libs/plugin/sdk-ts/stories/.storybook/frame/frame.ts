import type { Decorator } from '@storybook/vue3-vite'
import WindowFrame from './WindowFrame.vue'
import type { NotePath } from './vault'

/** The icons a tab of the window is drawn with. */
export type TabIcon =
  'note' | 'book' | 'plex' | 'agent' | 'audio' | 'logs' | 'search' | 'settings' | 'deck'

export interface FrameTab {
  readonly title: string
  readonly icon: TabIcon
}

/**
 * Where a story is drawn in a stand-in of the application window, as `parameters.frame`:
 *
 * - `tab`: in the tab `tab`, in front of the tabs `otherTabs`, with the note in a pane `before` or `after` it;
 * - `files`: in place of the file tree;
 * - `title-bar`: in place of the bar;
 * - `status-bar`: below the panes, where the window has no bar of its own;
 * - `title-status`: in the status slot at the end of the title bar.
 *
 * The window around it holds the file tree of the Physics vault, with the row `selected` chosen, unless
 * `files` is false, and the note `notePath` in the panes a story does not take. `selected` is that note
 * when not given, and the folder of the chosen row is drawn open.
 */
export interface Frame {
  readonly place: 'tab' | 'files' | 'title-bar' | 'title-status' | 'status-bar'
  readonly tab?: FrameTab
  readonly otherTabs?: readonly FrameTab[]
  readonly note?: 'before' | 'after'
  readonly notePath?: NotePath
  readonly share?: number
  readonly selected?: string
  readonly files?: boolean
  readonly title?: string
}

/** Draws a story whose parameters name a `frame` inside the window, and any other story as it is. */
export const withWindowFrame: Decorator = (story, context) => {
  const frame = context.parameters['frame'] as Frame | undefined
  if (!frame) return { components: { story }, template: '<story />' }
  return {
    components: { story, WindowFrame },
    setup: () => ({ frame }),
    template: '<WindowFrame :frame="frame"><story /></WindowFrame>',
  }
}
