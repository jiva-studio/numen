/**
 * The contrast of text against what it is drawn on, as WCAG 2 measures it.
 *
 * The colours of a page are rarely flat: a ground can be translucent, and an
 * ancestor can be dimmed as a whole. So the colour a reader sees is resolved
 * down the chain of ancestors, each translucent ground laid over what lies
 * beneath it and each `opacity` blending its subtree with that backdrop.
 *
 * Nothing here needs a browser: the page's own colour parser is passed in, so
 * the arithmetic is tested in jsdom and the browser supplies only the parser.
 */

/** A colour with its coverage, each channel from 0 to 255 and the coverage from 0 to 1. */
export interface Rgba {
  readonly r: number
  readonly g: number
  readonly b: number
  readonly a: number
}

/** An opaque colour. */
export type Rgb = Pick<Rgba, 'r' | 'g' | 'b'>

/** Reads any CSS colour a computed style can hold. */
export type ParseColour = (css: string) => Rgba

/** WCAG 2 minimum for text below the large-text size. */
export const TEXT_RATIO = 4.5

/** WCAG 2 minimum for large text and for the parts of a control that are not text. */
export const LARGE_RATIO = 3

const channel = (value: number): number => {
  const unit = value / 255
  return unit <= 0.03928 ? unit / 12.92 : ((unit + 0.055) / 1.055) ** 2.4
}

/** Relative luminance of an opaque colour, from 0 to 1. */
export function getLuminance(colour: Rgb): number {
  return 0.2126 * channel(colour.r) + 0.7152 * channel(colour.g) + 0.0722 * channel(colour.b)
}

/** The ratio of the lighter colour's luminance to the darker's, each lifted by 0.05. */
export function getContrastRatio(first: Rgb, second: Rgb): number {
  const [light, dark] = [getLuminance(first), getLuminance(second)].sort((a, b) => b - a)
  return ((light ?? 0) + 0.05) / ((dark ?? 0) + 0.05)
}

/** A colour laid over an opaque ground. */
export function layOver(ground: Rgb, top: Rgba): Rgb {
  return {
    r: ground.r + (top.r - ground.r) * top.a,
    g: ground.g + (top.g - ground.g) * top.a,
    b: ground.b + (top.b - ground.b) * top.a,
  }
}

/** What a subtree drawn at an opacity looks like over its backdrop. */
export function blend(backdrop: Rgb, drawn: Rgb, opacity: number): Rgb {
  return layOver(backdrop, { ...drawn, a: opacity })
}

/** The two colours a reader tells apart. */
export interface Resolved {
  readonly text: Rgb
  readonly ground: Rgb
}

/** The ancestors of an element from the outermost to the element itself. */
const getChain = (element: Element): Element[] => {
  const chain: Element[] = []
  for (let at: Element | null = element; at; at = at.parentElement) chain.unshift(at)
  return chain
}

/**
 * The text colour and the ground behind it, as drawn.
 *
 * A ground made of an image cannot be resolved to one colour, so it is refused
 * and not guessed at.
 */
export function resolveColours(
  element: Element,
  parse: ParseColour,
  canvas: Rgb,
  read: (target: Element) => CSSStyleDeclaration = (target) => getComputedStyle(target),
): Resolved {
  const chain = getChain(element)
  const settle = (index: number, backdrop: Rgb): Resolved => {
    const target = chain[index]
    if (!target) throw new Error('no element to resolve')
    const style = read(target)
    if (style.backgroundImage && style.backgroundImage !== 'none')
      throw new Error(`${describe(target)} is drawn on an image, which has no one colour`)
    const ground = layOver(backdrop, parse(style.backgroundColor || 'transparent'))
    const opacity = Number.parseFloat(style.opacity || '1')
    const inner: Resolved =
      index === chain.length - 1
        ? { text: layOver(ground, parse(style.color)), ground }
        : settle(index + 1, ground)
    return {
      text: blend(backdrop, inner.text, opacity),
      ground: blend(backdrop, inner.ground, opacity),
    }
  }
  return settle(0, canvas)
}

/** The contrast ratio of an element's text against its ground. */
export function measureText(element: Element, parse: ParseColour, canvas: Rgb): number {
  const { text, ground } = resolveColours(element, parse, canvas)
  return getContrastRatio(text, ground)
}

/**
 * The contrast of an element's own fill against what it sits on: the part of a
 * control that is not text, such as a switch's thumb on its track.
 */
export function measureFill(element: Element, parse: ParseColour, canvas: Rgb): number {
  const parent = element.parentElement
  if (!parent) throw new Error('a fill is measured against the element it sits on')
  const on = resolveColours(parent, parse, canvas).ground
  const own = resolveColours(element, parse, canvas).ground
  return getContrastRatio(own, on)
}

/** A short name for an element in a failure message. */
export function describe(element: Element): string {
  const classes = (element.getAttribute('class') ?? '').trim().split(/\s+/).filter(Boolean)
  return `${element.localName}${classes.length ? `.${classes.slice(0, 2).join('.')}` : ''}`
}

/** One element and what its contrast came to. */
export interface Reading {
  readonly name: string
  readonly ratio: number
}

/** The fewest a reading may be, and the elements that fall short of it. */
export function getShortfalls(readings: readonly Reading[], least: number): Reading[] {
  return readings.filter((reading) => reading.ratio < least)
}

/** Whether a colour is the accent, or the accent a hover has moved a little. */
export function isAccent(colour: Rgb, accent: Rgb, tolerance = 30): boolean {
  return (
    Math.abs(colour.r - accent.r) <= tolerance &&
    Math.abs(colour.g - accent.g) <= tolerance &&
    Math.abs(colour.b - accent.b) <= tolerance
  )
}

/** Whether an element holds text of its own, as opposed to only holding elements that do. */
export function hasOwnText(element: Element): boolean {
  return Array.from(element.childNodes).some(
    (node) => node.nodeType === 3 && (node.textContent ?? '').trim() !== '',
  )
}
