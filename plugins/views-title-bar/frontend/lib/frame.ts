import type { TitleBarViewProps } from '@numen/plugin-sdk'

/**
 * Who draws the window buttons and where: the bar at the right (`windows`), the bar as three
 * circles at the left (`mac`), or the system at the left (`native-mac`), where the bar keeps room
 * for them.
 */
export type ControlsStyle = 'windows' | 'mac' | 'native-mac'

/** The values of the setting `buttons`: where the bar draws the window buttons on Linux. */
export type ButtonsSetting = 'windows' | 'mac'

/** The key of the setting that places the window buttons on Linux. */
export const BUTTONS_SETTING = 'buttons'

/** A value of the setting `buttons`; a missing or unknown value is the Windows one. */
export function readButtonsSetting(value: unknown): ButtonsSetting {
  return value === 'mac' ? 'mac' : 'windows'
}

/**
 * Windows draws the buttons at the right to keep Snap Layouts, and macOS keeps the system buttons.
 * Linux draws them where the setting `buttons` says.
 */
export function getControls(
  platform: TitleBarViewProps['platform'],
  buttons: ButtonsSetting,
): ControlsStyle {
  if (platform === 'darwin') return 'native-mac'
  return platform === 'linux' ? buttons : 'windows'
}

/** The names of the window buttons, shown as their tooltips. */
export interface WindowControlsWords {
  readonly minimize: string
  readonly maximize: string
  readonly restore: string
  readonly close: string
}
