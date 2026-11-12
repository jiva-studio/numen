/**
 * Browser DOM, visibility, and platform adapters.
 */

export function getUserAgent(): string {
  return globalThis.navigator.userAgent
}

/** Returns the viewport size in CSS pixels. */
export function getViewportSize(): { width: number; height: number } {
  return { width: globalThis.innerWidth, height: globalThis.innerHeight }
}

/** Returns the screen position of the window's top-left corner. Wayland does not report it; both values are 0 there. */
export function getWindowOrigin(): { x: number; y: number } {
  return { x: globalThis.screenX, y: globalThis.screenY }
}
