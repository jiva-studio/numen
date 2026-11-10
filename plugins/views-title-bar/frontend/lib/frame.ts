import type { AppInfo, JsonValue } from '@numen/plugin-sdk'

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
export function readButtonsSetting(value: JsonValue | undefined): ButtonsSetting {
  return value === 'mac' || value === 'none' ? value : 'windows'
}

/** The key of the setting that hides the bar while the window is in full screen. */
export const HIDE_IN_FULLSCREEN_SETTING = 'hideInFullscreen'

/** A value of the setting `hideInFullscreen`; only `true` turns it on. */
export function readHideInFullscreenSetting(value: JsonValue | undefined): boolean {
  return value === true
}

/** Whether the bar is drawn: not while the setting hides it and the window is in full screen. */
export function isBarShown(hideInFullscreen: boolean, isFullscreen: boolean): boolean {
  return !(hideInFullscreen && isFullscreen)
}

/**
 * macOS keeps the system buttons whatever the setting says. Windows and Linux draw them where the
 * setting `buttons` says, or not at all.
 */
export function getControls(platform: AppInfo['platform'], buttons: ButtonsSetting): ControlsStyle {
  return platform === 'darwin' ? 'native-mac' : buttons
}

/** The width of the three window buttons the bar draws at its end in the Windows style, in pixels. */
const WINDOWS_CONTROLS_WIDTH = 132

/** The width of the three circles the bar draws at its start in the mac style, in pixels. */
const MAC_CIRCLES_WIDTH = 76

/** The width the bar keeps at its start for the system buttons of macOS, in pixels. */
const MAC_INSET_WIDTH = 80

/** The widths the window buttons take at the start and at the end of the bar, in pixels. */
export function getControlsRoom(
  controls: ControlsStyle,
  isFullscreen: boolean,
): { readonly lead: number; readonly trail: number } {
  if (controls === 'windows') return { lead: 0, trail: WINDOWS_CONTROLS_WIDTH }
  if (controls === 'mac') return { lead: MAC_CIRCLES_WIDTH, trail: 0 }
  return { lead: isFullscreen ? 0 : MAC_INSET_WIDTH, trail: 0 }
}

/**
 * The room the end of the bar keeps clear of the rounded corner of the window, as a CSS length. The
 * radius is the variable `--numen-window-radius` the window sets on its root (none on Windows and
 * Linux); the status entries are about 20 pixels high, centred in the bar, so their end can come
 * within the radius less 8 pixels of the edge.
 */
export const CORNER_INSET = 'calc(var(--numen-window-radius, 0px) - 8px)'

/** The names of the window buttons, shown as their tooltips. */
export interface WindowControlsWords {
  readonly minimize: string
  readonly maximize: string
  readonly restore: string
  readonly close: string
}
