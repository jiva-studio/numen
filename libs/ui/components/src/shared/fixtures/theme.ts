/**
 * Helpers for stories rendered with the dark theme. Storybook test fixture
 * only; it is not shipped.
 *
 * The toolbar sets the theme as `color-scheme` on the root element, which is what
 * `light-dark()` in the tokens reads, and names it in `data-theme` there, so a story checks the active token set
 * there.
 */
import { expect } from 'storybook/test'

/** The globals that render a story with the dark token set. */
export const DARK = { theme: 'dark' } as const

/** Fails unless the page containing the story uses the dark tokens. */
export async function expectDark(canvas: HTMLElement): Promise<void> {
  const root = canvas.ownerDocument.documentElement
  await expect(root.dataset['theme']).toBe('dark')
}
