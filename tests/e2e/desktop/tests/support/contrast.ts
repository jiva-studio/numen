/**
 * The contrast of what a page draws, as the page computes it.
 *
 * A measurement resolves the text colour of an element and the ground under it, the grounds of its
 * ancestors laid over one another and the opacity of each blended in, so a translucent ground or a
 * dimmed control is measured as it is seen. The page supplies the computed styles and reads colours
 * of any space into red, green, blue and coverage; the arithmetic is done here.
 */
import { expect, type Locator } from 'playwright/test'

/** The ratio WCAG 2 asks of text below the large-text size. */
export const TEXT_RATIO = 4.5

/** The ratio WCAG 2 asks of large text and of a mark that is not text. */
export const MARK_RATIO = 3

/** One element, what it says, the colours it is drawn in and how far apart they stand. */
export interface Measurement {
  readonly name: string
  readonly text: string
  readonly ground: string
  readonly ratio: number
  /** The least the element must reach: text and marks are asked for different ratios. */
  readonly least: number
}

/** One element of the chain from the page to the measured element, as the page computes it. */
interface Layer {
  readonly ground: string
  readonly opacity: number
  readonly color: string
  readonly isImage: boolean
}

/** An element as the page describes it: its layers from the outermost, and how it is named. */
interface Described {
  readonly name: string
  readonly hasText: boolean
  readonly layers: readonly Layer[]
}

type Rgb = readonly [number, number, number]
type Rgba = readonly [number, number, number, number]

/** The computed styles of the elements a selector finds in a scope, the scope itself among them. */
const describeElements = (scope: Locator, selector: string): Promise<Described[]> =>
  scope.evaluate((root, query) => {
    const found = [root, ...root.querySelectorAll(query)].filter((one) => one.matches(query))
    const described = []
    for (const element of found) {
      if (element.getClientRects().length === 0) continue
      if (element.hasAttribute('disabled') || element.getAttribute('aria-disabled') === 'true')
        continue
      for (const motion of element.getAnimations({ subtree: true })) motion.finish()
      const layers = []
      for (let node: Element | null = element; node; node = node.parentElement) {
        const style = getComputedStyle(node)
        layers.unshift({
          ground: style.backgroundColor,
          opacity: Number.parseFloat(style.opacity),
          color: style.color,
          isImage: style.backgroundImage !== 'none',
        })
      }
      const words = (element.getAttribute('aria-label') ?? element.textContent ?? '')
        .replace(/\s+/g, ' ')
        .trim()
      const classes = String(element.className).split(' ')[0]
      described.push({
        name: `${element.localName}${classes ? `.${classes}` : ''} "${words.slice(0, 32)}"`,
        hasText: (element.textContent ?? '').trim() !== '',
        layers,
      })
    }
    return described
  }, selector)

/** Any computed colour, whatever space it is written in, as red, green, blue and coverage. */
const parseColours = (scope: Locator, colours: readonly string[]): Promise<Record<string, Rgba>> =>
  scope.evaluate((_root, list) => {
    const paint = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
    if (!paint) throw new Error('no canvas to read a colour on')
    const paintOver = (colour: string, ground: string): number[] => {
      for (const fill of [ground, colour]) {
        paint.fillStyle = fill
        paint.fillRect(0, 0, 1, 1)
      }
      return [...paint.getImageData(0, 0, 1, 1).data]
    }
    const parsed: Record<string, [number, number, number, number]> = {}
    for (const css of list) {
      const [rb = 0, gb = 0, bb = 0] = paintOver(css, '#000')
      const [rw = 0, gw = 0, bw = 0] = paintOver(css, '#fff')
      const a = Math.min(1, Math.max(0, 1 - (rw - rb + (gw - gb) + (bw - bb)) / 765))
      parsed[css] = a < 0.002 ? [0, 0, 0, 0] : [rb / a, gb / a, bb / a, a]
    }
    return parsed
  }, colours)

/** A colour laid over an opaque ground. */
const layOver = (ground: Rgb, [r, g, b, a]: Rgba): Rgb => [
  ground[0] + (r - ground[0]) * a,
  ground[1] + (g - ground[1]) * a,
  ground[2] + (b - ground[2]) * a,
]

const getLuminance = (colour: Rgb): number => {
  const [r, g, b] = colour.map((value) => {
    const unit = value / 255
    return unit <= 0.03928 ? unit / 12.92 : ((unit + 0.055) / 1.055) ** 2.4
  }) as [number, number, number]
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

const toHex = (colour: Rgb): string =>
  `#${colour.map((value) => Math.round(value).toString(16).padStart(2, '0')).join('')}`

/** The text and ground colours at the end of a chain of layers, each opacity blended with what is behind. */
const resolveLayers = (
  layers: readonly Layer[],
  parsed: Record<string, Rgba>,
  canvas: Rgb,
): { text: Rgb; ground: Rgb } => {
  const settle = (index: number, backdrop: Rgb): { text: Rgb; ground: Rgb } => {
    const layer = layers[index]
    if (!layer) throw new Error('no layer to resolve')
    if (layer.isImage) throw new Error('a ground that is an image has no one colour')
    const ground = layOver(backdrop, parsed[layer.ground] ?? [0, 0, 0, 0])
    const inner =
      index === layers.length - 1
        ? { text: layOver(ground, parsed[layer.color] ?? [0, 0, 0, 1]), ground }
        : settle(index + 1, ground)
    const blend = (drawn: Rgb): Rgb => layOver(backdrop, [...drawn, layer.opacity])
    return { text: blend(inner.text), ground: blend(inner.ground) }
  }
  return settle(0, canvas)
}

/**
 * The measurements of the elements a selector finds in a scope, the scope itself among them.
 * Elements that are not drawn, and controls that are disabled, which WCAG leaves out, are skipped.
 */
export const measureContrasts = async (
  scope: Locator,
  selector: string,
): Promise<Measurement[]> => {
  const described = await describeElements(scope, selector)
  const colours = new Set(['Canvas'])
  for (const one of described)
    for (const layer of one.layers) [layer.ground, layer.color].forEach((c) => colours.add(c))
  const parsed = await parseColours(scope, [...colours])
  const [r = 255, g = 255, b = 255] = parsed['Canvas'] ?? []
  const canvas: Rgb = [r, g, b]
  return described.map((one) => {
    const { text, ground } = resolveLayers(one.layers, parsed, canvas)
    const [light = 0, dark = 0] = [getLuminance(text), getLuminance(ground)].sort((a, b) => b - a)
    return {
      name: one.name,
      text: toHex(text),
      ground: toHex(ground),
      ratio: (light + 0.05) / (dark + 0.05),
      least: one.hasText ? TEXT_RATIO : MARK_RATIO,
    }
  })
}

/** The measurements below what each is asked for, written out with both colours. */
export const listShortfalls = (measurements: readonly Measurement[]): string[] =>
  measurements
    .filter((one) => one.ratio < one.least)
    .map(
      (one) =>
        `${one.name}: ${one.text} on ${one.ground} is ${one.ratio.toFixed(2)}:1, below ${one.least}:1`,
    )

/**
 * Asserts that at least `count` elements of the selector are drawn under the scope, and that every
 * one reaches its ratio. The count keeps a selector that finds nothing from passing for a page
 * that is readable.
 */
export const expectReadable = async (
  scope: Locator,
  selector: string,
  count: number,
): Promise<Measurement[]> => {
  const measurements = await measureContrasts(scope, selector)
  expect(measurements.length, `${selector} found too few drawn elements`).toBeGreaterThanOrEqual(
    count,
  )
  expect(listShortfalls(measurements)).toEqual([])
  return measurements
}
