/** Upright boxes in the plex's own coordinates: the area a node occupies, and the area a run of a line occupies. */
import { createPointOnCurve, type PlacedEdge } from '../../edge'
import type { PlacedNode, Position } from '../../node'

/** An upright box in the plex's own coordinates. */
export interface Box {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** How many points along a run the box around it is drawn from. */
const RUN_SAMPLES = 8

export const isOverlapping = (one: Box, other: Box): boolean =>
  one.minX < other.maxX && other.minX < one.maxX && one.minY < other.maxY && other.minY < one.maxY

export const getBox = (node: PlacedNode, apart: number): Box => ({
  minX: node.x - node.width / 2 - apart,
  minY: node.y - node.height / 2 - apart,
  maxX: node.x + node.width / 2 + apart,
  maxY: node.y + node.height / 2 + apart,
})

/**
 * The area a title occupies, step by step along the run it is set on.
 *
 * A title set across the plex is a ribbon, not a rectangle: a single box would
 * cover the whole plex and leave no placement for a line crossing near it.
 */
export interface BoxSpan {
  readonly from: number
  readonly to: number
}

export function getTitleBoxes(
  boxAt: (from: number, to: number) => Box,
  span: BoxSpan,
  step: number,
): Box[] {
  const { from, to } = span
  const boxes: Box[] = []
  for (let offset = from; offset < to; offset += step) {
    boxes.push(boxAt(offset, Math.min(offset + step, to)))
  }
  return boxes
}

/**
 * The box a run of the line fills. The words follow the line, so the run is
 * sampled along it and each sample carries the depth of a line of type across
 * the line, which is where the letters stand.
 */
export function runBoxes(edge: PlacedEdge, depth: number): (from: number, to: number) => Box {
  const axisEdge = createPointOnCurve(edge)
  const deep = depth / 2

  return (from, to) => {
    const run: Position[] = []
    for (let sample = 0; sample <= RUN_SAMPLES; sample += 1) {
      run.push(axisEdge(from + ((to - from) * sample) / RUN_SAMPLES))
    }

    let minX = Infinity
    let minY = Infinity
    let maxX = -Infinity
    let maxY = -Infinity

    for (const [sample, point] of run.entries()) {
      const back = run[Math.max(sample - 1, 0)]!
      const on = run[Math.min(sample + 1, run.length - 1)]!
      const span = Math.hypot(on.x - back.x, on.y - back.y) || 1
      const acrossX = (Math.abs(on.y - back.y) / span) * deep
      const acrossY = (Math.abs(on.x - back.x) / span) * deep

      minX = Math.min(minX, point.x - acrossX)
      minY = Math.min(minY, point.y - acrossY)
      maxX = Math.max(maxX, point.x + acrossX)
      maxY = Math.max(maxY, point.y + acrossY)
    }

    return { minX, minY, maxX, maxY }
  }
}
