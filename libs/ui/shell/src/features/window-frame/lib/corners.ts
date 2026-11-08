import type { WindowPlatform } from '../model/useWindowFrame'

/**
 * The corner radius of a macOS window with a hidden title bar before macOS 26, in points. Every
 * window has the same radius there.
 */
export const MAC_RADIUS_EARLIER = 10

/**
 * The corner radius of a macOS 26 window, in points. Windows with a toolbar are rounder than windows
 * with a title bar only, and macOS documents no value, so this is the larger one: content kept
 * clear of it is never clipped.
 */
export const MAC_RADIUS_TAHOE = 26

/** The first macOS version with the larger corner radius. */
export const MAC_TAHOE_MAJOR = 26

/** The name of the CSS variable on the document root that holds the radius, as a length. */
export const WINDOW_RADIUS_VARIABLE = '--numen-window-radius'

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
  return macMajor !== undefined && macMajor < MAC_TAHOE_MAJOR ? MAC_RADIUS_EARLIER : MAC_RADIUS_TAHOE
}
