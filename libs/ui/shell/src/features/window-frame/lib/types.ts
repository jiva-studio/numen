/**
 * Who draws the window buttons and where: the page at the right (`windows`), the page as three
 * circles at the left (`mac`), or the system at the left (`native-mac`), where the page keeps room
 * for them.
 */
export type ControlsStyle = 'windows' | 'mac' | 'native-mac'

/** The names of the window buttons, shown as their tooltips. */
export interface WindowControlsWords {
  readonly minimise: string
  readonly maximise: string
  readonly restore: string
  readonly close: string
}

export const WINDOW_CONTROLS_WORDS: WindowControlsWords = {
  minimise: 'Minimise',
  maximise: 'Maximise',
  restore: 'Restore',
  close: 'Close',
}
