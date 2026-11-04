/**
 * Browser DOM, visibility, and platform adapters.
 */

export function getUserAgent(): string {
  return globalThis.navigator.userAgent
}

/** The size of the page's viewport, in CSS pixels. */
export function getViewportSize(): { width: number; height: number } {
  return { width: globalThis.innerWidth, height: globalThis.innerHeight }
}

/** Where the window's top-left corner is on the screen; 0 where the platform does not say, as on Wayland. */
export function getWindowOrigin(): { x: number; y: number } {
  return { x: globalThis.screenX, y: globalThis.screenY }
}
