/**
 * Browser DOM, visibility, and platform adapters.
 */

export function isDocumentHidden(): boolean {
  return globalThis.document.hidden
}

export function getUserAgent(): string {
  return globalThis.navigator.userAgent
}
