/**
 * What a colour a browser hands back actually comes to. Storybook's furniture;
 * it ships to nobody.
 *
 * A computed colour may be named in any space, and a translucent one is only a
 * colour once there is something behind it. Both are settled by painting the
 * ground and the colour over it and reading the pixel back.
 */

/** A colour laid over the ground under it, as red, green and blue. */
export function laid(colour: string, ground = '#000'): readonly [number, number, number] {
  const paint = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
  if (!paint) throw new Error('no canvas to read a colour on')
  for (const fill of [ground, colour]) {
    paint.fillStyle = fill
    paint.fillRect(0, 0, 1, 1)
  }
  const [red = 0, green = 0, blue = 0] = paint.getImageData(0, 0, 1, 1).data
  return [red, green, blue]
}

/** How light a colour is over the ground under it, from 0 to 255. */
export function lightness(colour: string, ground = '#000'): number {
  const [red, green, blue] = laid(colour, ground)
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

/**
 * A real pointer put over an element, so the browser's own `:hover` applies.
 * The events a test library synthesises leave it alone.
 */
export async function hovered(element: Element): Promise<void> {
  const context = await import('vitest/browser')
  await context.userEvent.hover(element)
}
