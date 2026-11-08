import type { ControlsStyle, TitleBarAction, TitleChoice } from '../lib/types'
import type { TitleBarWords } from '../lib/words'

export interface TitleBarProps {
  appName: string
  vaultName?: string
  controls?: ControlsStyle
  isMaximised?: boolean
  actions?: readonly TitleBarAction[]
  words?: Partial<TitleBarWords>
  choices?: readonly TitleChoice[]
}
