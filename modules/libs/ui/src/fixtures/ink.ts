/**
 * Where a browser actually put the ink, and how thick it is. Storybook's
 * furniture; it ships to nobody.
 *
 * A story that asks whether two things are drawn alike cannot ask the DOM: a
 * box is not ink, and a mark can sit anywhere inside its box. So the drawing is
 * painted again onto a canvas, large, and read back a pixel at a time.
 *
 * Thickness is the middle, over the ink, of the shortest run through a pixel in
 * any of the four directions — which is a stroke's own width for a stroke lying
 * flat, upright, or at either diagonal. Counting along one direction alone
 * reads a diagonal half again as thick as it is.
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

const painting = (): CanvasRenderingContext2D => {
  const canvas = document.createElement('canvas')
  canvas.width = GRAIN
  canvas.height = GRAIN
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) throw new Error('no canvas to measure on')
  return ctx
}

/** The lines the ink on a painting reaches, and how thick it is, in its pixels. */
const painted = (ctx: CanvasRenderingContext2D): { top: number; bottom: number; run: number } => {
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

/**
 * The ink of a run of text.
 *
 * Where a browser puts the baseline of an inline box is half the leading below
 * its top, so the ink a font reports around that baseline is where the letter
 * stands on the page.
 */
export const letterInk = (span: Element): Ink => {
  const box = span.getBoundingClientRect()
  const style = getComputedStyle(span)
  const size = parseFloat(style.fontSize)
  const text = span.textContent ?? ''

  const ctx = painting()
  ctx.font = `${size}px ${style.fontFamily}`
  const metrics = ctx.measureText(text)
  const lead = (box.height - (metrics.fontBoundingBoxAscent + metrics.fontBoundingBoxDescent)) / 2
  const baseline = box.top + lead + metrics.fontBoundingBoxAscent

  // Large, so a stem one pixel wide on the page is measured in tens of them.
  const scale = GRAIN / 3 / size
  ctx.font = `${size * scale}px ${style.fontFamily}`
  ctx.textBaseline = 'middle'
  ctx.fillStyle = '#000'
  ctx.fillText(text, 4, GRAIN / 2)

  return {
    top: baseline - metrics.actualBoundingBoxAscent,
    bottom: baseline + metrics.actualBoundingBoxDescent,
    middle: baseline - (metrics.actualBoundingBoxAscent - metrics.actualBoundingBoxDescent) / 2,
    stroke: painted(ctx).run / scale,
  }
}

/** The ink of a drawing, painted again large and read back into the box it fills. */
export const markInk = async (mark: SVGElement): Promise<Ink> => {
  const box = mark.getBoundingClientRect()
  const copy = mark.cloneNode(true) as SVGElement
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
  const ctx = painting()
  ctx.drawImage(image, 0, 0, GRAIN, GRAIN)

  const ink = painted(ctx)
  const down = (at: number): number => box.top + (at / GRAIN) * box.height
  return {
    top: down(ink.top),
    bottom: down(ink.bottom),
    middle: down((ink.top + ink.bottom) / 2),
    stroke: (ink.run / GRAIN) * box.width,
  }
}
