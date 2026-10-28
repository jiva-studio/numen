/**
 * Browser DOM, visibility, platform, and clock adapters.
 */

export function getNow(): number {
  return Date.now()
}

export function isDocumentHidden(): boolean {
  return globalThis.document.hidden
}

export function getUserAgent(): string {
  return globalThis.navigator.userAgent
}
