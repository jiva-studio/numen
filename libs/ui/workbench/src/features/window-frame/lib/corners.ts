/** The operating system of the window, as `runtime.GOOS` names it. */
export type WindowPlatform = 'darwin' | 'windows' | 'linux'

/**
 * The corner radius of a macOS window with a hidden title bar before macOS 26, in points. Every
 * window has the same radius there.
 */
const MAC_RADIUS_EARLIER = 10

/**
 * The corner radius of a macOS 26 window with a hidden title bar, in points, measured on a screenshot
 * of macOS 26.6.2 (14 to 16). Windows with a toolbar are rounder; macOS documents no value.
 */
const MAC_RADIUS_TAHOE = 16

/** The first macOS version with the larger corner radius. */
const MAC_TAHOE_MAJOR = 26

/**
 * The name of the CSS variable on the document root that holds the width of the hairline macOS
 * draws along the top edge of the window, as a length: the bar moves its row down by it.
 */
export const WINDOW_EDGE_VARIABLE = '--numen-window-edge'

/** The width in points of the top hairline of a macOS window. */
const MAC_TOP_EDGE = 1

/** The name of the CSS variable on the document root that holds the radius, as a length. */
export const WINDOW_RADIUS_VARIABLE = '--numen-window-radius'

/** The width of the top hairline of the window in points: 1 on macOS, none in full screen and elsewhere. */
export function getWindowEdge(platform: WindowPlatform, isFullscreen: boolean): number {
  return platform === 'darwin' && !isFullscreen ? MAC_TOP_EDGE : 0
}

/** The major number of a version like "26.0" or "15.6.1"; undefined when it is not a number. */
export function readMajorVersion(version: string | undefined): number | undefined {
  const major = Number.parseInt(version ?? '', 10)
  return Number.isNaN(major) ? undefined : major
}

/**
 * The corner radius of the window, in points: none on Windows and Linux, none on macOS in full
 * screen, and on macOS the radius of its version. An unknown version gets the larger radius.
 */
export function getWindowRadius(
  platform: WindowPlatform,
  macMajor: number | undefined,
  isFullscreen: boolean,
): number {
  if (platform !== 'darwin' || isFullscreen) return 0
  return macMajor !== undefined && macMajor < MAC_TAHOE_MAJOR
    ? MAC_RADIUS_EARLIER
    : MAC_RADIUS_TAHOE
}
