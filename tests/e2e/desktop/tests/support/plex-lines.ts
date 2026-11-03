import type { Page } from 'playwright/test'
import { getLabelGaps, type LabelGap, type SampledLine } from './plex-labels.js'

export interface Point {
  readonly x: number
  readonly y: number
}

export interface Box {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** The line between two drawn notes: its end at each note, its middle, its labels and its heads, on the screen. */
export interface LineGeometry {
  readonly ends: Readonly<Record<string, Point>>
  readonly middle: Point
  readonly labels: Readonly<Record<string, Point>>
  /** The box each label is drawn in, on the screen. */
  readonly boxes: Readonly<Record<string, Box>>
  /** Where the letters of each label are against the line, measured the same way for every label. */
  readonly gaps: Readonly<Record<string, LabelGap>>
  /**
   * How much each label is read in the direction from the first named note to the second: 1 along
   * the line that way, -1 along it the other way, 0 across it.
   */
  readonly reading: Readonly<Record<string, number>>
  readonly arrows: readonly Point[]
  /** The curve sampled on the screen. */
  readonly line: SampledLine
}

/**
 * Reads the line between two notes off the drawing. The line is the one whose ends lie nearest
 * the two titles; each label and each head is measured where the browser drew it.
 */
const readDrawnLine = (page: Page, names: readonly [string, string]) =>
  page.evaluate((names) => {
    const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
      (one) => one.getBoundingClientRect().width > 0,
    )!
    const centre = (box: DOMRect) => ({ x: box.x + box.width / 2, y: box.y + box.height / 2 })
    const distance = (one: { x: number; y: number }, other: { x: number; y: number }) =>
      Math.hypot(one.x - other.x, one.y - other.y)
    const titles = names.map((name) => {
      const title = [...svg.querySelectorAll('.plex__title')].find(
        (one) => one.textContent?.trim() === name,
      )!
      return centre(title.getBoundingClientRect())
    })

    const lines = [...svg.querySelectorAll<SVGPathElement>('.plex__edge')].map((path) => {
      const screen = path.getScreenCTM()!
      const length = path.getTotalLength()
      const getPoint = (offset: number) => {
        const point = path.getPointAtLength(offset).matrixTransform(screen)
        return { x: point.x, y: point.y }
      }
      const points = Array.from({ length: Math.ceil(length / 2) + 1 }, (_, step) =>
        getPoint(Math.min(2 * step, length)),
      )
      return { start: getPoint(0), end: getPoint(length), middle: getPoint(length / 2), points }
    })
    const getFit = (line: (typeof lines)[number]) =>
      Math.min(
        distance(line.start, titles[0]!) + distance(line.end, titles[1]!),
        distance(line.end, titles[0]!) + distance(line.start, titles[1]!),
      )
    const line = lines.reduce((best, one) => (getFit(one) < getFit(best) ? one : best))
    const isStartAtFirst =
      distance(line.start, titles[0]!) + distance(line.end, titles[1]!) <=
      distance(line.end, titles[0]!) + distance(line.start, titles[1]!)

    const labels: Record<string, { x: number; y: number }> = {}
    const boxes: Record<string, Box> = {}
    const directions: Record<string, number> = {}
    const way = isStartAtFirst ? 1 : -1
    for (const letters of svg.querySelectorAll('.plex__edge-label--letters')) {
      const box = letters.getBoundingClientRect()
      const words = letters.textContent?.trim() ?? ''
      labels[words] = centre(box)
      boxes[words] = { x: box.x, y: box.y, width: box.width, height: box.height }

      // Measure against the line where it is nearest the label, taken from the first note to the second.
      const middle = centre(box)
      const nearest = line.points.reduce(
        (best, point, step) =>
          distance(point, middle) < distance(line.points[best]!, middle) ? step : best,
        0,
      )
      const back = line.points[Math.max(nearest - 1, 0)]!
      const on = line.points[Math.min(nearest + 1, line.points.length - 1)]!
      const run = { x: (on.x - back.x) * way, y: (on.y - back.y) * way }
      const size = Math.hypot(run.x, run.y) || 1
      const turn = ((letters as SVGTextContentElement).getRotationOfChar(0) * Math.PI) / 180
      directions[words] = (Math.cos(turn) * run.x + Math.sin(turn) * run.y) / size
    }
    const arrows = [...svg.querySelectorAll('.plex__edge-arrow')].map((head) =>
      centre(head.getBoundingClientRect()),
    )
    return {
      ends: {
        [names[0]]: isStartAtFirst ? line.start : line.end,
        [names[1]]: isStartAtFirst ? line.end : line.start,
      },
      middle: line.middle,
      labels,
      boxes,
      points: line.points,
      way,
      reading: directions,
      arrows,
    }
  }, names)

/** The line between two notes of the plex, with the gap of each label against it. */
export const readPlexLine = async (
  page: Page,
  names: readonly [string, string],
): Promise<LineGeometry> => {
  const { points, way, ...drawn } = await readDrawnLine(page, names)
  const line: SampledLine = { points, way: way as 1 | -1 }
  return { ...drawn, line, gaps: await getLabelGaps(page, line) }
}

export const getDistance = (one: Point, other: Point) =>
  Math.hypot(one.x - other.x, one.y - other.y)
