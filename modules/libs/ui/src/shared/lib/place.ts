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
  /** Where the span it stands beside begins, and where it ends. */
  readonly from: number
  readonly to: number
  /** How large the thing being placed turned out to be. */
  readonly size: number
  /** How far the area it is placed in reaches. */
  readonly room: number
  /** Kept clear of that area's edges, so nothing sits flush against them. */
  readonly margin: number
  /** Left between it and the span it stands beside. */
  readonly gap: number
}

/**
 * A thing runs on from the far end of the span it stands beside. Where the far
 * edge is nearer than its own size it runs back from the near end, and either
 * way it is brought inside the edges it may touch. Larger than the area it is
 * placed in, it sits at the near edge.
 */
export const getPlaceBeside = ({ from, to, size, room, margin, gap }: AxisPlacement): number => {
  const on = to + gap
  const back = from - gap - size
  const start = on + size + margin <= room || back < margin ? on : back
  return Math.max(margin, Math.min(start, room - size - margin))
}
