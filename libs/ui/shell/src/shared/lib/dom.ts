/**
 * Browser DOM, visibility, and platform adapters.
 */

export function getUserAgent(): string {
  return globalThis.navigator.userAgent
}
