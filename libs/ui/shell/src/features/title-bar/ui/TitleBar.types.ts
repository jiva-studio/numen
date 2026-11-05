import type { ControlsStyle, TitleBarAction, TitleChoice } from '../lib/types'
import type { TitleBarWords } from '../lib/words'

export interface TitleBarProps {
  appName: string
  /** The vault in the window; empty where none is open. */
  vaultName?: string
  controls?: ControlsStyle
  isMaximised?: boolean
  /** Sorted as they are to be drawn. */
  actions?: readonly TitleBarAction[]
  words?: Partial<TitleBarWords>
  /** What the title offers to switch to. A bar with none draws the title as plain text. */
  choices?: readonly TitleChoice[]
}
