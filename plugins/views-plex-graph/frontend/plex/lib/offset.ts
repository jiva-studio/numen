import type { NodeHeadings } from './inside'

/** The furthest offset the window on the headings may be scrolled to, counted in headings. */
export const getFurthestOffset = (suspended: NodeHeadings): number =>
  Math.max(0, suspended.headings.length - suspended.visibleCount)
