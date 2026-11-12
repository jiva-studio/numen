/**
 * A real pointer put over an element, so the browser's own `:hover` applies.
 * The events a test library synthesizes leave it alone. Storybook support code;
 * not shipped.
 */
export async function hoverOver(element: Element): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.hover(element)
}
