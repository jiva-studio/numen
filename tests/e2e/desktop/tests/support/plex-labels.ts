import type { Page } from 'playwright/test'

/** A point on the screen. */
export interface ScreenPoint {
  readonly x: number
  readonly y: number
}

/** The curve of a line sampled on the screen, and which way its samples run against the first note to the second. */
export interface SampledLine {
  readonly points: readonly ScreenPoint[]
  /** 1 where the samples run from the first note to the second, -1 where they run back. */
  readonly way: 1 | -1
}

/** Where the letters of one label are against the line, in pixels on the screen. */
export interface LabelGap {
  /**
   * How farthest the nearest edge of the letters is from the line, normal it: the ink from the top
   * of a capital to the bottom of a descender, measured from the baseline the browser set.
   * Positive when the letters are clear of the line, negative when the line runs through them.
   */
  readonly gap: number
  /** On which side of the line the letters are: positive on the right looking from the first note to the second. */
  readonly side: number
}

/**
 * Measures every label of the visible plex against one line, in the same way for a line with one
 * label and for a line with two: from the baseline the browser gave the first letter, moved by the
 * half x-height that centring the text on its path puts between them, so the distance does not
 * depend on the rotation of the text or on the engine's box for it.
 */
export const getLabelGaps = (
  page: Page,
  line: SampledLine,
): Promise<Readonly<Record<string, LabelGap>>> =>
  page.evaluate(({ points, way }) => {
    const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
      (one) => one.getBoundingClientRect().width > 0,
    )!
    const context = document.createElement('canvas').getContext('2d')!
    const gaps: Record<string, { gap: number; side: number }> = {}
    for (const letters of svg.querySelectorAll<SVGTextElement>('.plex__edge-label--letters')) {
      const style = getComputedStyle(letters)
      context.font = `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`
      const ink = context.measureText('Hxg')
      const screen = letters.getScreenCTM()!
      const scale = Math.hypot(screen.a, screen.b)
      const origin = letters.getStartPositionOfChar(0).matrixTransform(screen)
      const turn = (letters.getRotationOfChar(0) * Math.PI) / 180
      // The way up the page of the letters, on the screen.
      const up = { x: Math.sin(turn), y: -Math.cos(turn) }

      let nearest = 0
      let best = Infinity
      points.forEach((point, step) => {
        const apart = Math.hypot(point.x - origin.x, point.y - origin.y)
        if (apart < best) {
          best = apart
          nearest = step
        }
      })
      const here = points[nearest]!
      const back = points[Math.max(nearest - 1, 0)]!
      const on = points[Math.min(nearest + 1, points.length - 1)]!
      const run = { x: (on.x - back.x) * way, y: (on.y - back.y) * way }
      const size = Math.hypot(run.x, run.y) || 1
      const normal = { x: -run.y / size, y: run.x / size }
      const lean = up.x * normal.x + up.y * normal.y
      // The text is set with its middle on the path: its baseline is half the height of an x below it.
      const drop = (context.measureText('x').actualBoundingBoxAscent / 2) * scale
      const base = (origin.x - here.x) * normal.x + (origin.y - here.y) * normal.y - lean * drop
      const top = base + lean * ink.actualBoundingBoxAscent * scale
      const bottom = base - lean * ink.actualBoundingBoxDescent * scale
      const closest = Math.min(top, bottom)
      const farthest = Math.max(top, bottom)
      const side = closest > 0 ? 1 : farthest < 0 ? -1 : 0
      const gap = side > 0 ? closest : side < 0 ? -farthest : -Math.min(farthest, -closest)
      gaps[letters.textContent?.trim() ?? ''] = { gap, side }
    }
    return gaps
  }, line)

/** The most a label may be from its line: the letters hug it. */
export const MAX_LABEL_GAP = 4

/** The most the gaps of two labels may differ by: every label is at the one distance. */
export const MAX_GAP_DIFFERENCE = 1.5

/** Whether the letters are beside the line, touching neither it nor the line's other side, and close to it. */
export const isHugging = (entry: LabelGap | undefined): boolean =>
  entry !== undefined && entry.gap > 0 && entry.gap <= MAX_LABEL_GAP

/** Whether two labels are at the same distance from their lines. */
export const isAtSameGap = (one: LabelGap | undefined, other: LabelGap | undefined): boolean =>
  one !== undefined && other !== undefined && Math.abs(one.gap - other.gap) < MAX_GAP_DIFFERENCE
