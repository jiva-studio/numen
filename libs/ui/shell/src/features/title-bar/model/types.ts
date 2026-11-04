import type { Component } from 'vue'

/** One button of the bar, as the bar draws it. The window decides what it runs. */
export interface TitleBarAction {
  readonly id: string
  /** What the button is called in its tooltip and in the menu of the actions that do not fit. */
  readonly title: string
  readonly icon: Component
  /** Actions that share a group are drawn together, with a separator between groups. */
  readonly group?: string
  /** The shortcut as it is shown to the person, such as `Ctrl+Shift+E`. */
  readonly shortcut?: string
}

/**
 * Who draws the window controls: the bar at the right (`windows`), the bar as three circles at the
 * left (`mac`), or the system at the left (`native-mac`), where the bar keeps room for it.
 */
export type ControlsStyle = 'windows' | 'mac' | 'native-mac'
