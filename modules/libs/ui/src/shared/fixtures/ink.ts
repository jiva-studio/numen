/**
 * Where a browser actually put a drawing's ink, and how thick it is.
 * Storybook's furniture; it ships to nobody.
 *
 * A story that asks whether two drawings are drawn alike cannot ask the DOM: a
 * box is not ink, and a drawing can sit anywhere inside its box. So it is
 * painted again onto a canvas, large, and read back a pixel at a time.
 *
 * Thickness is the middle, over the ink, of the shortest run through a pixel in
 * any of the four directions — which is a stroke's own width for a stroke lying
 * flat, upright, or at either diagonal. Counting along one direction alone
 * reads a diagonal half again as thick as it is.
 *
 * Only drawings are measured here. Where a font puts ink inside its em box is
 * that font's business and differs from machine to machine, so nothing a test
 * holds is anchored to it.
 */

/** Where a run of ink is, down the page, and how thick it is. In CSS pixels. */
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

/** The lines the ink on a painting reaches, and how thick it is, in its pixels. */
const readInk = (ctx: CanvasRenderingContext2D): { top: number; bottom: number; run: number } => {
  const data = ctx.getImageData(0, 0, GRAIN, GRAIN).data
  const on = (x: number, y: number): boolean =>
    x >= 0 && y >= 0 && x < GRAIN && y < GRAIN && data[(y * GRAIN + x) * 4 + 3]! > 128

  const inked: [number, number][] = []
  let top = -1
  let bottom = -1
  for (let y = 0; y < GRAIN; y += 1) {
    for (let x = 0; x < GRAIN; x += 1) {
      if (!on(x, y)) continue
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
  const across: number[] = []
  for (const [x, y] of inked) {
    let least = Infinity
    for (const [dx, dy, step] of WAYS) {
      let run = 1
      for (let out = 1; on(x + dx * out, y + dy * out); out += 1) run += 1
      for (let out = 1; on(x - dx * out, y - dy * out); out += 1) run += 1
      least = Math.min(least, run * step)
    }
    across.push(least)
  }
  across.sort((one, other) => one - other)
  return { top, bottom: bottom + 1, run: across[across.length >> 1] ?? 0 }
}

/** The ink of a drawing, painted again large and read back into the box it fills. */
export const drawingInk = async (drawing: SVGElement): Promise<Ink> => {
  const box = drawing.getBoundingClientRect()
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

  const ink = readInk(ctx)
  const down = (at: number): number => box.top + (at / GRAIN) * box.height
  return {
    top: down(ink.top),
    bottom: down(ink.bottom),
    middle: down((ink.top + ink.bottom) / 2),
    stroke: (ink.run / GRAIN) * box.width,
  }
}
