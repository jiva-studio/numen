import type { Decorator } from '@storybook/vue3-vite'
import WindowFrame from './WindowFrame.vue'

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
 * - `tab`: in the tab `tab`, in front of the tabs `behind`, with the note in a pane `before` or `after` it;
 * - `files`: in place of the file tree;
 * - `title-bar`, `status-bar`: in place of the bar;
 * - `title-status`: in the status slot at the end of the title bar.
 *
 * The window around it holds the file tree of the Physics vault, with the row `selected` chosen, unless
 * `files` is false, and a note in the panes a story does not take.
 */
export interface Frame {
  readonly place: 'tab' | 'files' | 'title-bar' | 'title-status' | 'status-bar'
  readonly tab?: FrameTab
  readonly behind?: readonly FrameTab[]
  readonly note?: 'before' | 'after'
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
