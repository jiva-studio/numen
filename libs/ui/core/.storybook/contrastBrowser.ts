/**
 * The contrast helpers as a story runs them: the page's own colour parser, the
 * transitions of a hover finished, and a sweep of a story for text on the accent.
 */
import { expect } from 'storybook/test'
import type { StoryCheck } from './check'
import {
  LARGE_RATIO,
  TEXT_RATIO,
  describe,
  getContrastRatio,
  getShortfalls,
  hasOwnText,
  isAccent,
  measureFill,
  measureText,
  resolveColours,
  type ParseColour,
  type Reading,
  type Rgb,
  type Rgba,
} from './contrast'

/** The one canvas colours are read on, made when the first is. */
let paint: CanvasRenderingContext2D | null = null

function paintOver(colour: string, ground: string): readonly [number, number, number] {
  paint ??= document.createElement('canvas').getContext('2d', { willReadFrequently: true })
  if (!paint) throw new Error('no canvas to read a colour on')
  for (const fill of [ground, colour]) {
    paint.fillStyle = fill
    paint.fillRect(0, 0, 1, 1)
  }
  const [red = 0, green = 0, blue = 0] = paint.getImageData(0, 0, 1, 1).data
  return [red, green, blue]
}

/** What each colour string came to, since a page repeats a few colours down a long tree. */
const parsed = new Map<string, Rgba>()

/**
 * Any computed colour, whatever space it is written in, with its coverage. It
 * is painted once over black and once over white; the distance the two pixels
 * keep is what the colour lets through.
 */
export const parseColour: ParseColour = (css) => {
  const known = parsed.get(css)
  if (known) return known
  const [rb, gb, bb] = paintOver(css, '#000')
  const [rw, gw, bw] = paintOver(css, '#fff')
  const letThrough = (rw - rb + (gw - gb) + (bw - bb)) / (3 * 255)
  const a = Math.min(1, Math.max(0, 1 - letThrough))
  const colour: Rgba =
    a < 0.002 ? { r: 0, g: 0, b: 0, a: 0 } : { r: rb / a, g: gb / a, b: bb / a, a }
  parsed.set(css, colour)
  return colour
}

/** What the page is drawn on where nothing draws a ground. */
export function getCanvas(): Rgb {
  const probe = document.createElement('div')
  probe.style.backgroundColor = 'Canvas'
  document.documentElement.append(probe)
  const { r, g, b } = parseColour(getComputedStyle(probe).backgroundColor)
  probe.remove()
  return { r, g, b }
}

/** Runs every transition and animation of an element to its end, so a state is read as it settles. */
export function finishMotion(element: Element): void {
  for (const motion of element.getAnimations({ subtree: true })) motion.finish()
}

/** The contrast of an element's text against its ground, after its transitions end. */
export function getTextRatio(element: Element): number {
  finishMotion(element)
  return measureText(element, parseColour, getCanvas())
}

/** The contrast of an element's fill against what it sits on, after its transitions end. */
export function getFillRatio(element: Element): number {
  finishMotion(element)
  if (element.parentElement) finishMotion(element.parentElement)
  return measureFill(element, parseColour, getCanvas())
}

/** The text colour and the ground of an element, for a message. */
export function getColours(element: Element): { text: string; ground: string } {
  const { text, ground } = resolveColours(element, parseColour, getCanvas())
  const hex = (c: Rgb) =>
    `#${[c.r, c.g, c.b].map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')}`
  return { text: hex(text), ground: hex(ground) }
}

/** Prints one reading as a line the run can be read back from, and returns it. */
export function note(kind: string, name: string, ratio: number): number {
  console.info(`CONTRAST|${kind}|${name}|${ratio.toFixed(2)}`)
  return ratio
}

/** Fails unless the element's text reaches the ratio; the failure names both colours. */
export async function expectText(
  element: Element,
  kind: string,
  name: string,
  minimum: number = TEXT_RATIO,
): Promise<number> {
  const ratio = note(kind, name, getTextRatio(element))
  const { text, ground } = getColours(element)
  await expect(
    ratio,
    `${kind} ${name}: ${text} on ${ground} is ${ratio.toFixed(2)}:1, below ${minimum}:1`,
  ).toBeGreaterThanOrEqual(minimum)
  return ratio
}

/** Fails unless the element's fill reaches the ratio against what it sits on. */
export async function expectFill(
  element: Element,
  kind: string,
  name: string,
  minimum: number = LARGE_RATIO,
): Promise<number> {
  const ratio = note(kind, name, getFillRatio(element))
  await expect(
    ratio,
    `${kind} ${name}: fill is ${ratio.toFixed(2)}:1 against what it sits on, below ${minimum}:1`,
  ).toBeGreaterThanOrEqual(minimum)
  return ratio
}

function isHidden(element: Element): boolean {
  for (let node: Element | null = element; node; node = node.parentElement) {
    const style = getComputedStyle(node)
    if (style.visibility === 'hidden' || Number.parseFloat(style.opacity) === 0) return true
  }
  return false
}

function getToken(root: Element, token: string): Rgba {
  const probe = document.createElement('div')
  probe.style.backgroundColor = `var(${token})`
  root.append(probe)
  const colour = parseColour(getComputedStyle(probe).backgroundColor)
  probe.remove()
  return colour
}

/**
 * Asserts that every element under the root that holds text of its own reaches the ratio against
 * the ground under it. Text in the hushed colour is the voice of a caption; the token's own pair
 * with the surface is measured and printed under `<kind> hushed`, and held to no ratio here.
 */
export async function expectAllText(
  root: Element,
  kind: string,
  minimum: number = TEXT_RATIO,
): Promise<number> {
  finishMotion(root)
  const canvas = getCanvas()
  const hushed = getToken(root, '--numen-hushed')
  let count = 0
  for (const element of [root, ...root.querySelectorAll('*')]) {
    if (!hasOwnText(element)) continue
    if (isHidden(element)) continue
    const { text, ground } = resolveColours(element, parseColour, canvas)
    const ratio = getContrastRatio(text, ground)
    const words = (element.textContent ?? '').trim()
    const name = `${describe(element)} "${words.slice(0, 24)}"`
    // One glyph, such as the cross that closes a tab, is a mark and not a line of text.
    const least = words.length === 1 ? Math.min(minimum, LARGE_RATIO) : minimum
    count += 1
    if (isAccent(text, hushed, 2)) {
      note(`${kind} hushed`, name, ratio)
      continue
    }
    note(kind, name, ratio)
    await expect(
      ratio,
      `${kind} ${name}: ${ratio.toFixed(2)}:1, below ${least}:1`,
    ).toBeGreaterThanOrEqual(least)
  }
  return count
}

/**
 * Every element of a story that holds text on the accent, or text in the
 * accent, with the ratio of its text to its ground. A story that draws a solid
 * control, or a tinted label, cannot leave the text unreadable, whichever
 * component drew it.
 */
export function readAccentText(root: Element): Reading[] {
  const canvas = getCanvas()
  const accentProbe = document.createElement('div')
  accentProbe.style.backgroundColor = 'var(--numen-accent)'
  root.append(accentProbe)
  const accent = parseColour(getComputedStyle(accentProbe).backgroundColor)
  accentProbe.remove()

  const styles = new Map<Element, CSSStyleDeclaration>()
  const read = (target: Element): CSSStyleDeclaration => {
    const known = styles.get(target) ?? getComputedStyle(target)
    styles.set(target, known)
    return known
  }
  const readings: Reading[] = []
  for (const element of root.querySelectorAll('*')) {
    if (!hasOwnText(element)) continue
    let resolved
    try {
      resolved = resolveColours(element, parseColour, canvas, read)
    } catch {
      continue
    }
    if (!isAccent(resolved.ground, accent) && !isAccent(resolved.text, accent)) continue
    readings.push({
      name: describe(element),
      ratio: getContrastRatio(resolved.text, resolved.ground),
    })
  }
  return readings
}

/** The sweep as an `afterEach` of a corpus: fails a story whose text on the accent is unreadable. */
export const accentCheck: StoryCheck = async (context) => {
  if (typeof window === 'undefined' || !('__vitest_browser__' in window)) return
  const faults = getShortfalls(readAccentText(context.canvasElement), TEXT_RATIO)
  if (faults.length)
    throw new Error(
      `${context.id}: text on the accent below ${TEXT_RATIO}:1: ${faults
        .map((fault) => `${fault.name} ${fault.ratio.toFixed(2)}:1`)
        .join('; ')}`,
    )
}
