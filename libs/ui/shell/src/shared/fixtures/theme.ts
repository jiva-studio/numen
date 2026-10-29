/**
 * What a story rendered dark depends on. Storybook scaffolding; it is not
 * shipped.
 *
 * A theme is chosen by `color-scheme` on the root, which is what `light-dark()`
 * in the tokens reads, so that is where a story queries which set of colours it
 * was rendered in.
 */
import { expect } from 'storybook/test'

/** The globals a story carries to be drawn in the dark set of tokens. */
export const DARK = { theme: 'dark' } as const

/** Fail unless the page around the story is standing on the dark tokens. */
export async function expectDark(canvas: HTMLElement): Promise<void> {
  const root = canvas.ownerDocument.documentElement
  await expect(getComputedStyle(root).colorScheme).toBe('dark')
}
