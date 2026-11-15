import type { WindowPlatform } from '../lib/corners'
import { MAC_BUTTONS_INSET } from '../lib/sizes'

/** The room the system buttons take in the top corners of the window, in pixels. */
export interface FrameEdge {
  readonly leftInset: number
  readonly rightInset: number
}

/**
 * The room the system buttons take in the top corners: the macOS buttons at the left, except in full
 * screen, where macOS shows none. Windows and Linux have no buttons without a title bar.
 */
export function getWindowEdge(platform: WindowPlatform, isFullscreen: boolean): FrameEdge {
  const hasMacButtons = platform === 'darwin' && !isFullscreen
  return { leftInset: hasMacButtons ? MAC_BUTTONS_INSET : 0, rightInset: 0 }
}
