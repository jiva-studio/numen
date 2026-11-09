/**
 * A real press of the pointer, struck by the browser that draws the story. Storybook test fixture
 * only; it is not shipped. The events a test library synthesises do not show how WebKit picks the
 * target of a click when the node under the press is replaced.
 */
export async function pressOn(element: Element): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.click(element)
}

export async function pressKey(key: string): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.keyboard(key)
}
