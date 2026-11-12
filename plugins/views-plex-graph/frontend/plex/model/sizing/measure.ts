/**
 * How wide a box has to be to hold its title, and how far a label's words run
 * along their line.
 *
 * The plex measures its own text: the type a title is set in is the type it is
 * arranged from. The answer comes at once, while the arrangement is worked out.
 */
import { doc, hasDocument } from '@numen/ui'
import { computed, onScopeDispose, shallowRef, type Ref } from 'vue'
import type { LabelType } from '../../lib/edge'
import type { PlexNode } from '../../../types'

/** The width a node's box needs, padding included. */
type Measure = (node: PlexNode) => number

/** What a plex measures its own text with. */
export interface PlexMetrics {
  readonly node: Measure
  readonly label: (label: string) => number
  readonly heading: (text: string) => number
  readonly labelType: LabelType
}

/** The type a plex is drawn with, and the gap between the things set in one row, in pixels. */
export interface PlexType {
  readonly font: string
  readonly labelFont: string
  readonly gap: number
}

/**
 * What a title is set in beside its text: the padding on each side, and the widths of what
 * is drawn before it, each a gap away from the next.
 */
export interface TitleRoom {
  readonly padding: number
  readonly beside: readonly number[]
}

/**
 * The declarations a title is drawn under, carried by a box the document
 * resolves them against: a token written in any unit is returned in pixels. The
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
  'column-gap:var(--numen-node-gap)',
].join(';')

/** The word beside it, set as a label on a line is. */
const LABEL = 'font-size:var(--numen-edge-label-size)'

/** Something with a width, in each of the two sizes. */
const SAMPLE = 'Hxg'

export interface Probe {
  readonly box: HTMLElement
  readonly label: HTMLElement
}

const parsePixels = (value: string): number => {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : 0
}

/**
 * The measurers for the titles a plex draws, taking their type and lengths from the document and rebuilt whenever that type changes.
 * The first measurement is taken before anything is drawn, and a measurement equal to the last changes nothing.
 * `room` is the padding of a title and the space reserved beside it for what the caller draws there.
 * Undefined where there is no canvas to measure against; the arrangement then draws every box at its widest.
 */
export function useTitleWidths(
  room: () => TitleRoom,
  readType: (probe: Probe) => PlexType = getType,
): Ref<PlexMetrics | undefined> {
  const probe = openProbe()
  if (!probe) return shallowRef<PlexMetrics | undefined>(undefined)

  const type = shallowRef(readType(probe))

  if (typeof ResizeObserver === 'undefined') {
    probe.box.remove()
  } else {
    const observer = new ResizeObserver(() => {
      const now = readType(probe)
      if (!sameType(now, type.value)) type.value = now
    })
    observer.observe(probe.box, { box: 'border-box' })

    onScopeDispose(() => {
      observer.disconnect()
      probe.box.remove()
    }, true)
  }

  return computed(() => createMetrics(type.value, room()))
}

function openProbe(): Probe | null {
  if (!hasDocument() || !doc.body) return null

  const box = doc.createElement('div')
  box.setAttribute('aria-hidden', 'true')
  box.style.cssText = PROBE

  // Stryker disable next-line StringLiteral: the mutated value is "Stryker was here!", which is not a declaration, so the style engine drops it and the word keeps the box's own type, as with the empty string.
  const title = word('')
  const label = word(LABEL)
  box.append(title, label)
  doc.body.append(box)

  return { box, label }
}

function word(style: string): HTMLElement {
  const span = doc.createElement('span')
  span.style.cssText = style
  span.textContent = SAMPLE
  return span
}

function getType(probe: Probe): PlexType {
  const box = getComputedStyle(probe.box)
  const label = getComputedStyle(probe.label)
  const family = box.fontFamily || 'sans-serif'

  return {
    font: `${box.fontSize} ${family}`,
    labelFont: `${label.fontSize} ${family}`,
    gap: parsePixels(box.columnGap),
  }
}

const sameType = (one: PlexType, other: PlexType): boolean =>
  one.font === other.font && one.labelFont === other.labelFont && one.gap === other.gap

function createMetrics(type: PlexType, { padding, beside }: TitleRoom): PlexMetrics | undefined {
  const context = getMeasuringContext()
  if (!context) return undefined

  const title = textWidths(context, type.font)
  const label = textWidths(context, type.labelFont)

  const room = beside.reduce((sum, width) => sum + width + type.gap, 2 * padding)

  return {
    node: (node) => Math.ceil(title(node.title) + room),
    label: (words) => Math.ceil(label(words)),
    heading: (text) => Math.ceil(label(text) + 2 * padding),
    labelType: getLabelType(context, type.labelFont),
  }
}

function getLabelType(context: CanvasRenderingContext2D, font: string): LabelType {
  context.font = font
  const size = parsePixels(font)
  return {
    ascent: context.measureText('H').actualBoundingBoxAscent || 0.72 * size,
    descent: context.measureText('g').actualBoundingBoxDescent || 0.22 * size,
    xHeight: context.measureText('x').actualBoundingBoxAscent || 0.5 * size,
  }
}

function textWidths(context: CanvasRenderingContext2D, font: string): (text: string) => number {
  const widths = new Map<string, number>()

  return (text) => {
    const recognized = widths.get(text)
    if (recognized !== undefined) return recognized

    context.font = font
    const width = context.measureText(text).width
    widths.set(text, width)
    return width
  }
}

function getMeasuringContext(): CanvasRenderingContext2D | null {
  if (typeof CanvasRenderingContext2D === 'undefined') return null
  try {
    return doc.createElement('canvas').getContext('2d')
  } catch {
    // A window that will not make one has none, which is what the check above
    // already answers for; measuring is not offered either way.
    return null
  }
}
