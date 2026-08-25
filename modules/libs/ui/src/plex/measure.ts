/**
 * How wide a box has to be to hold its title.
 *
 * The plex measures its own text: the type a title is set in is the type it is
 * arranged from. The answer comes at once, while the arrangement is worked out.
 */
import type { PlexNode } from './model'

/** The width a node's box needs, padding included. */
export type Measure = (node: PlexNode) => number

/** Whitespace in a token is however it was written; a font shorthand is one line. */
const oneLine = (value: string): string => value.trim().replace(/\s+/g, ' ')

const numberOf = (value: string, fallback: number): number => {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

/**
 * A measurer for the titles a plex draws, taking its type and padding from the
 * root, which is where a theme writes them. The root stands before anything is
 * mounted, so the first arrangement is measured like every one after it.
 *
 * Nothing where there is no canvas to measure against — jsdom, or a page
 * rendered on a server — and the arrangement then draws every box at its widest.
 */
export function titleWidths(): Measure | undefined {
  if (typeof document === 'undefined') return undefined

  const context = measuringContext()
  if (!context) return undefined

  const styles = getComputedStyle(document.documentElement)
  const size = oneLine(styles.getPropertyValue('--numen-font-size')) || '13px'
  const family = oneLine(styles.getPropertyValue('--numen-font-sans')) || 'sans-serif'
  const padding = numberOf(styles.getPropertyValue('--numen-node-padding'), 10)

  context.font = `${size} ${family}`

  // Keyed by the string: one measurement for each distinct title.
  const widths = new Map<string, number>()

  return (node: PlexNode): number => {
    const known = widths.get(node.title)
    if (known !== undefined) return known

    const width = Math.ceil(context.measureText(node.title).width + 2 * padding)
    widths.set(node.title, width)
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
    return null
  }
}
