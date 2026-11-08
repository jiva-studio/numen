import type { AppInfo } from '@numen/plugin-sdk'

/**
 * Who draws the window buttons and where: the bar at the right (`windows`), the bar as three
 * circles at the left (`mac`), no one (`none`), or the system at the left (`native-mac`), where the
 * bar keeps room for them.
 */
export type ControlsStyle = 'windows' | 'mac' | 'none' | 'native-mac'

/** The values of the setting `buttons`: where the bar draws the window buttons, or `none`. */
export type ButtonsSetting = 'windows' | 'mac' | 'none'

/** The key of the setting that places the window buttons. */
export const BUTTONS_SETTING = 'buttons'

/** A value of the setting `buttons`; a missing or unknown value is the Windows one. */
export function readButtonsSetting(value: unknown): ButtonsSetting {
  return value === 'mac' || value === 'none' ? value : 'windows'
}

/**
 * macOS keeps the system buttons whatever the setting says. Windows and Linux draw them where the
 * setting `buttons` says, or not at all.
 */
export function getControls(platform: AppInfo['platform'], buttons: ButtonsSetting): ControlsStyle {
  return platform === 'darwin' ? 'native-mac' : buttons
}

/** The names of the window buttons, shown as their tooltips. */
export interface WindowControlsWords {
  readonly minimize: string
  readonly maximize: string
  readonly restore: string
  readonly close: string
}
