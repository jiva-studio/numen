/**
 * The RGB value a browser-computed color resolves to. Storybook fixture only;
 * it is not shipped.
 *
 * A computed color may be expressed in any color space, and a translucent one is
 * only a color once there is a backdrop behind it. Both are resolved by painting
 * the backdrop and then the color over it and reading the pixel back.
 */

function resolveColor(color: string, ground = '#000'): readonly [number, number, number] {
  const paint = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
  if (!paint) throw new Error('no canvas to read a color on')
  for (const fill of [ground, color]) {
    paint.fillStyle = fill
    paint.fillRect(0, 0, 1, 1)
  }
  const [red = 0, green = 0, blue = 0] = paint.getImageData(0, 0, 1, 1).data
  return [red, green, blue]
}

/** How light a color is over the backdrop, from 0 to 255. */
export function lightness(color: string, ground = '#000'): number {
  const [red, green, blue] = resolveColor(color, ground)
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

/**
 * A real pointer put over an element, so the browser's own `:hover` applies.
 * The events a test library synthesizes leave it alone.
 */
export async function hoverOver(element: Element): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.hover(element)
}
