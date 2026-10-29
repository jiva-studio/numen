/**
 * Helpers for stories rendered with the dark theme. Storybook support code;
 * not shipped.
 *
 * A theme is chosen by `color-scheme` on the root, which is what `light-dark()`
 * in the tokens reads. A story reads the set of colours it is rendered in from
 * that attribute.
 */
import { expect } from 'storybook/test'

/** The globals a story carries to be drawn in the dark set of tokens. */
export const DARK = { theme: 'dark' } as const

/** Fail unless the page around the story is standing on the dark tokens. */
export async function expectDark(canvas: HTMLElement): Promise<void> {
  const root = canvas.ownerDocument.documentElement
  await expect(getComputedStyle(root).colorScheme).toBe('dark')
}
