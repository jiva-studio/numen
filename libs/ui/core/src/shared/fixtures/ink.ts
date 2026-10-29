/**
 * Measures the painted extent and the stroke thickness of an SVG drawing as the
 * browser renders it. Storybook fixture; not shipped.
 *
 * A story that compares two drawings cannot use DOM boxes: a box is not the
 * painted area, and a drawing can sit anywhere inside its box. The drawing is
 * repainted onto a large canvas and read back pixel by pixel.
 *
 * Thickness is the median, over the painted pixels, of the shortest run through
 * a pixel in any of the four directions. It equals the stroke width for a
 * horizontal, vertical or diagonal stroke. Counting along one direction alone
 * measures a diagonal stroke 1.5 times thicker than it is.
 *
 * Only drawings are measured. The painted position of a font inside its em box
 * differs between machines, so no test assertion depends on it.
 */

/** The vertical extent of the painted area and its stroke thickness. In CSS pixels. */
export interface Ink {
  readonly top: number
  readonly bottom: number
  readonly middle: number
  readonly stroke: number
}

/** How large a drawing is painted again to be measured, in pixels a side. */
const GRAIN = 240

const createCanvas = (): CanvasRenderingContext2D => {
  const canvas = document.createElement('canvas')
  canvas.width = GRAIN
  canvas.height = GRAIN
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) throw new Error('no canvas to measure on')
  return ctx
}

/** The first and last painted rows of a canvas and the stroke thickness, in canvas pixels. */
const readStroke = (
  ctx: CanvasRenderingContext2D,
): { top: number; bottom: number; run: number } => {
  const data = ctx.getImageData(0, 0, GRAIN, GRAIN).data
  const isInked = (x: number, y: number): boolean =>
    x >= 0 && y >= 0 && x < GRAIN && y < GRAIN && data[(y * GRAIN + x) * 4 + 3]! > 128

  const inked: [number, number][] = []
  let top = -1
  let bottom = -1
  for (let y = 0; y < GRAIN; y += 1) {
    for (let x = 0; x < GRAIN; x += 1) {
      if (!isInked(x, y)) continue
      inked.push([x, y])
      if (top < 0) top = y
      bottom = y
    }
  }
  if (!inked.length) return { top: 0, bottom: 0, run: 0 }

  const WAYS: readonly [number, number, number][] = [
    [1, 0, 1],
    [0, 1, 1],
    [1, 1, Math.SQRT2],
    [1, -1, Math.SQRT2],
  ]
  const acrossIndices: number[] = []
  for (const [x, y] of inked) {
    let least = Infinity
    for (const [dx, dy, step] of WAYS) {
      let run = 1
      for (let out = 1; isInked(x + dx * out, y + dy * out); out += 1) run += 1
      for (let out = 1; isInked(x - dx * out, y - dy * out); out += 1) run += 1
      least = Math.min(least, run * step)
    }
    acrossIndices.push(least)
  }
  acrossIndices.sort((one, other) => one - other)
  return { top, bottom: bottom + 1, run: acrossIndices[acrossIndices.length >> 1] ?? 0 }
}

import { measureElementRect } from '@/shared/lib/dom'

/** The painted area of a drawing, repainted large and scaled back to the box it fills. */
export const getDrawingStroke = async (drawing: SVGElement): Promise<Ink> => {
  const box = measureElementRect(drawing)
  const copy = drawing.cloneNode(true) as SVGElement
  copy.setAttribute('width', String(GRAIN))
  copy.setAttribute('height', String(GRAIN))
  copy.setAttribute('stroke', '#000')

  const image = new Image()
  await new Promise((ok, no) => {
    image.onload = ok
    image.onerror = no
    image.src =
      'data:image/svg+xml;charset=utf-8,' +
      encodeURIComponent(new XMLSerializer().serializeToString(copy))
  })
  const ctx = createCanvas()
  ctx.drawImage(image, 0, 0, GRAIN, GRAIN)

  const ink = readStroke(ctx)
  const getPageY = (pos: number): number => box.top + (pos / GRAIN) * box.height
  return {
    top: getPageY(ink.top),
    bottom: getPageY(ink.bottom),
    middle: getPageY((ink.top + ink.bottom) / 2),
    stroke: (ink.run / GRAIN) * box.width,
  }
}
