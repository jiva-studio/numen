/**
 * A text matcher for prose the thread draws one word per element: it finds the innermost element
 * whose whole text is the given one.
 */
export const byWholeText =
  (text: string) =>
  (_content: string, element: Element | null): boolean =>
    element?.textContent?.trim() === text &&
    !Array.from(element.children).some((child) => child.textContent?.trim() === text)
