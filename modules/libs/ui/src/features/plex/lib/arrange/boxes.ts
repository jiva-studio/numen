/** Upright boxes in the plex's own coordinates: the room a node takes, and the room a run of a line takes. */
import { rulerOf, type PlacedEdge } from '../edge'
import type { PlacedNode, Position } from '../node'

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
  one.minX < other.maxX &&
  other.minX < one.maxX &&
  one.minY < other.maxY &&
  other.minY < one.maxY

export const boxOf = (node: PlacedNode, apart: number): Box => ({
  minX: node.x - node.width / 2 - apart,
  minY: node.y - node.height / 2 - apart,
  maxX: node.x + node.width / 2 + apart,
  maxY: node.y + node.height / 2 + apart,
})

/**
 * The room a title takes up, step by step along the run it is set on.
 *
 * A title set across the picture is a ribbon and not a rectangle: the box
 * around the whole of a diagonal run stands over most of a quarter of the
 * picture, and a line crossing anywhere near it would find nowhere to be.
 */
export function ribbonOf(
  boxAt: (from: number, to: number) => Box,
  from: number,
  to: number,
  step: number,
): Box[] {
  const boxes: Box[] = []
  for (let at = from; at < to; at += step) {
    boxes.push(boxAt(at, Math.min(at + step, to)))
  }
  return boxes
}

/**
 * The box a run of the line fills. The words follow the line, so the run is
 * sampled along it and each sample carries the depth of a line of type across
 * the line, which is where the letters stand.
 */
export function runBoxes(edge: PlacedEdge, depth: number): (from: number, to: number) => Box {
  const along = rulerOf(edge)
  const deep = depth / 2

  return (from, to) => {
    const run: Position[] = []
    for (let sample = 0; sample <= RUN_SAMPLES; sample += 1) {
      run.push(along(from + ((to - from) * sample) / RUN_SAMPLES))
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
