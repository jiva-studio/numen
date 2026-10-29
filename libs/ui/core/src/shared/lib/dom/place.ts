/**
 * Where something standing over the page goes, one axis at a time. No DOM, no
 * measurement, no clock.
 *
 * The menu and the tooltip are placed by this, so the two answer one question
 * one way.
 */

/** A box on the screen, in pixels from its top left corner. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** What placing something beside a span takes, on one axis. */
export interface AxisPlacement {
  /** Start and end of the anchor span the element is placed beside. */
  readonly from: number
  readonly to: number
  /** How large the thing being placed turned out to be. */
  readonly size: number
  /** Extent of the container the element is placed in. */
  readonly room: number
  /** Minimum distance kept from the container's edges. */
  readonly margin: number
  /** Distance between the element and the anchor span. */
  readonly gap: number
}

/**
 * The element starts at the far end of the anchor span. When the space past the
 * far edge is smaller than its size, it starts before the near end instead, and
 * either way it is clamped inside the container margins. When it is larger than
 * the container, it sits at the near edge.
 */
export const getAdjacentPlace = ({ from, to, size, room, margin, gap }: AxisPlacement): number => {
  const on = to + gap
  const back = from - gap - size
  const start = on + size + margin <= room || back < margin ? on : back
  return Math.max(margin, Math.min(start, room - size - margin))
}
