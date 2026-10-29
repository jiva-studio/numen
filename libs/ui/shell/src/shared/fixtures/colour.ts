/**
 * The colour a browser computes for a CSS colour value, resolved to RGB.
 * Storybook support code; it is not shipped.
 *
 * A computed colour may be expressed in any colour space, and a translucent
 * colour has a value only when composited over a background. Both are resolved
 * by painting the background and the colour over it on a canvas and reading the
 * pixel back.
 */

/** A colour composited over the background, as red, green and blue. */
function resolveColour(colour: string, ground = '#000'): readonly [number, number, number] {
  const paint = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
  if (!paint) throw new Error('no canvas to read a colour on')
  for (const fill of [ground, colour]) {
    paint.fillStyle = fill
    paint.fillRect(0, 0, 1, 1)
  }
  const [red = 0, green = 0, blue = 0] = paint.getImageData(0, 0, 1, 1).data
  return [red, green, blue]
}

/** How light a colour is over the background, from 0 to 255. */
export function lightness(colour: string, ground = '#000'): number {
  const [red, green, blue] = resolveColour(colour, ground)
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

/**
 * A real pointer put over an element, so the browser's own `:hover` applies.
 * The events a test library synthesises leave it alone.
 */
export async function hoverOver(element: Element): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.hover(element)
}
