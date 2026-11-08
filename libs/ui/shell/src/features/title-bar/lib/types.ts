import type { Component } from 'vue'

/** One button of the bar, as the bar draws it. The window decides what it runs. */
export interface TitleBarAction {
  readonly id: string
  readonly title: string
  readonly icon: Component
  readonly group?: string
  readonly shortcut?: string
}

/**
 * One entry of the list the title opens. The window decides what the entries are and what choosing
 * one does; the bar only draws them.
 */
export interface TitleChoice {
  readonly id: string
  readonly label: string
  readonly isCurrent?: boolean
}

/**
 * Who draws the window controls: the bar at the right (`windows`), the bar as three circles at the
 * left (`mac`), or the system at the left (`native-mac`), where the bar keeps room for it.
 */
export type ControlsStyle = 'windows' | 'mac' | 'native-mac'
