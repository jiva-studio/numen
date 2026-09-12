/**
 * How wide a box has to be to hold its title, and how far a label's words run
 * along their line.
 *
 * The plex measures its own text: the type a title is set in is the type it is
 * arranged from. The answer comes at once, while the arrangement is worked out.
 */
import { computed, onScopeDispose, shallowRef, type Ref } from 'vue'
import type { PlexNode } from '../lib/node'

/** The width a node's box needs, padding included. */
export type Measure = (node: PlexNode) => number

/** What a plex measures its own text with. */
export interface PlexMetrics {
  readonly node: Measure
  /** The words of a label, which stand on a line and carry no padding. */
  readonly label: (label: string) => number
  /** The width a part's box needs for its words, padding included. */
  readonly part: (text: string) => number
  /** How deep one line of a label stands, across the line it is set on. */
  readonly labelDepth: number
}

/** The type and the lengths a plex is drawn with, in pixels. */
interface PlexType {
  /** A font shorthand, ready for a canvas. */
  readonly font: string
  readonly labelFont: string
  readonly padding: number
  readonly gap: number
}

/**
 * The declarations a title is drawn under, carried by a box the document
 * resolves them against: a token written in any unit comes back in pixels. The
 * box holds a word in each size a plex sets text in, so its own width answers a
 * change in any of the tokens it names.
 */
const PROBE = [
  'position:fixed',
  'inset-block-start:0',
  'inset-inline-start:0',
  'visibility:hidden',
  'pointer-events:none',
  'white-space:pre',
  'display:inline-flex',
  'font-family:var(--numen-font-sans)',
  'font-size:var(--numen-font-size)',
  'padding-inline:var(--numen-node-padding)',
  'column-gap:var(--numen-node-gap)',
].join(';')

/** The word beside it, set as a label on a line is. */
const LABEL = 'font-size:var(--numen-edge-label-size)'

/** Something with a width, in each of the two sizes. */
const SAMPLE = 'Hxg'

interface Probe {
  readonly box: HTMLElement
  readonly label: HTMLElement
}

/** A resolved length, which a document writes in pixels. */
const pixelsOf = (value: string): number => {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : 0
}

/**
 * The measurers for the titles a plex draws, taking their type and their
 * lengths from the document and remade whenever that type changes. The first
 * reading is taken before anything is drawn, and a reading that says what the
 * last one said changes nothing.
 *
 * `icon` is the room to keep beside a title for one the caller draws there.
 * Nothing at all where there is no canvas to measure against, and the
 * arrangement then draws every box at its widest.
 */
export function useTitleWidths(icon: () => number): Ref<PlexMetrics | undefined> {
  const probe = openProbe()
  if (!probe) return shallowRef<PlexMetrics | undefined>(undefined)

  const type = shallowRef(typeOf(probe))

  if (typeof ResizeObserver === 'undefined') {
    probe.box.remove()
  } else {
    const observer = new ResizeObserver(() => {
      const now = typeOf(probe)
      if (!sameType(now, type.value)) type.value = now
    })
    observer.observe(probe.box, { box: 'border-box' })

    onScopeDispose(() => {
      observer.disconnect()
      probe.box.remove()
    }, true)
  }

  return computed(() => measuresFor(type.value, icon()))
}

/** A box of the plex's own type, standing in the page. */
function openProbe(): Probe | null {
  if (typeof document === 'undefined' || !document.body) return null

  const box = document.createElement('div')
  box.setAttribute('aria-hidden', 'true')
  box.style.cssText = PROBE

  const title = word('')
  const label = word(LABEL)
  box.append(title, label)
  document.body.append(box)

  return { box, label }
}

function word(style: string): HTMLElement {
  const span = document.createElement('span')
  span.style.cssText = style
  span.textContent = SAMPLE
  return span
}

/** What the document makes of the tokens, read off the probe. */
function typeOf(probe: Probe): PlexType {
  const box = getComputedStyle(probe.box)
  const label = getComputedStyle(probe.label)
  const family = box.fontFamily || 'sans-serif'

  return {
    font: `${box.fontSize} ${family}`,
    labelFont: `${label.fontSize} ${family}`,
    padding: pixelsOf(box.paddingInlineStart),
    gap: pixelsOf(box.columnGap),
  }
}

const sameType = (one: PlexType, other: PlexType): boolean =>
  one.font === other.font &&
  one.labelFont === other.labelFont &&
  one.padding === other.padding &&
  one.gap === other.gap

function measuresFor(type: PlexType, icon: number): PlexMetrics | undefined {
  const context = measuringContext()
  if (!context) return undefined

  const title = textWidths(context, type.font)
  const label = textWidths(context, type.labelFont)

  // An icon stands a gap from the title, and both stand inside the padding.
  const room = 2 * type.padding + (icon > 0 ? icon + type.gap : 0)

  return {
    node: (node) => Math.ceil(title(node.title) + room),
    label: (words) => Math.ceil(label(words)),
    part: (text) => Math.ceil(label(text) + 2 * type.padding),
    labelDepth: lineDepth(context, type.labelFont),
  }
}

/**
 * How deep a line of one type stands: what the font gives its letters above the
 * baseline and below it. The size the type is set at where a platform reports
 * no such thing.
 */
function lineDepth(context: CanvasRenderingContext2D, font: string): number {
  context.font = font
  const line = context.measureText(SAMPLE)
  const deep = line.fontBoundingBoxAscent + line.fontBoundingBoxDescent
  return Math.ceil(deep) || pixelsOf(font)
}

/** Text in one type, each distinct string measured once. */
function textWidths(
  context: CanvasRenderingContext2D,
  font: string,
): (text: string) => number {
  const widths = new Map<string, number>()

  return (text) => {
    const known = widths.get(text)
    if (known !== undefined) return known

    context.font = font
    const width = context.measureText(text).width
    widths.set(text, width)
    return width
  }
}

/**
 * A canvas to measure text with, kept off the page. Nothing where the platform
 * has none of its own.
 */
function measuringContext(): CanvasRenderingContext2D | null {
  if (typeof CanvasRenderingContext2D === 'undefined') return null
  try {
    return document.createElement('canvas').getContext('2d')
  } catch {
    // A window that will not make one has none, which is what the check above
    // already answers for; measuring is not offered either way.
    return null
  }
}
