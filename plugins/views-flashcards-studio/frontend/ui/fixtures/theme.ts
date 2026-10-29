/**
 * What a story rendered in the dark theme requires. Storybook support code; it is
 * not shipped.
 *
 * A theme is chosen by `color-scheme` on the root, which is what `light-dark()`
 * in the tokens reads, so a story checks there which set of colours it was
 * rendered in.
 */
import { expect } from 'storybook/test'

/** The globals a story carries to be drawn in the dark set of tokens. */
export const DARK = { theme: 'dark' } as const

/** Fail unless the page around the story is standing on the dark tokens. */
export async function expectDark(canvas: HTMLElement): Promise<void> {
  const root = canvas.ownerDocument.documentElement
  await expect(getComputedStyle(root).colorScheme).toBe('dark')
}
