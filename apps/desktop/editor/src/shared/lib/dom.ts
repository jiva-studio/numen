/**
 * Browser window and navigator adapters for components.
 */

export function getUserAgent(): string {
  return navigator.userAgent
}

export function listenWindowPointerAndEscape(
  onPointerDown: (event: PointerEvent) => void,
  onKeyDown: (event: KeyboardEvent) => void,
): () => void {
  window.addEventListener('pointerdown', onPointerDown)
  window.addEventListener('keydown', onKeyDown)
  return () => {
    window.removeEventListener('pointerdown', onPointerDown)
    window.removeEventListener('keydown', onKeyDown)
  }
}
