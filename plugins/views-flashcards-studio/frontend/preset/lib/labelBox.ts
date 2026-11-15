import type { Position } from '@numen/ui'
import { PLOT_HEIGHT, PLOT_LEFT, PLOT_RIGHT, PLOT_TOP, PLOT_WIDTH } from './plot'
import { AXIS_LABEL_HEIGHT, AXIS_LABEL_LIFT, AXIS_LABEL_WIDTH, LABEL } from './labelSizes'

/** CSS inline position style for a plot label. */
export interface InlinePositionStyle {
  readonly insetInlineStart?: string
  readonly insetBlockStart?: string
  readonly translate?: string
  readonly [property: `--${string}`]: string | number | undefined
}
type CSSProperties = InlinePositionStyle

/** Bounding box of a label in plot units. */
export interface Box {
  readonly x: number
  readonly y: number
  readonly wide: number
  readonly high: number
}

/** Returns true when two label boxes do not overlap. */
export const areBoxesSeparate = (one: Box, two: Box): boolean =>
  one.x + one.wide <= two.x ||
  two.x + two.wide <= one.x ||
  one.y + one.high <= two.y ||
  two.y + two.high <= one.y

/** Returns true when an axis label at y does not overlap the curve or marks. */
export const isAxisLabelClear = (
  y: number,
  places: readonly Position[],
  marks: readonly (Position | null)[],
): boolean => {
  const isNear = (position: Position): boolean =>
    position.x <= PLOT_LEFT + AXIS_LABEL_WIDTH && Math.abs(position.y - y) <= AXIS_LABEL_HEIGHT
  if (marks.some((pos) => pos !== null && isNear(pos))) return false
  return !places.some(isNear)
}

export const getMarkNameStyle = (position: Position): CSSProperties => {
  let back = '-50%'
  if (position.x < PLOT_LEFT + LABEL) back = '0'
  else if (position.x > PLOT_RIGHT - LABEL) back = '-100%'
  return {
    insetInlineStart: `${(position.x / PLOT_WIDTH) * 100}%`,
    insetBlockStart: `${(Math.max(position.y - AXIS_LABEL_LIFT, PLOT_TOP) / PLOT_HEIGHT) * 100}%`,
    translate: `${back} -100%`,
  }
}

/** Returns the bounding box of a mark label. */
export const getNameBox = (position: Position): Box => {
  let back = LABEL
  if (position.x < PLOT_LEFT + LABEL) back = 0
  else if (position.x > PLOT_RIGHT - LABEL) back = LABEL * 2
  return {
    x: position.x - back,
    y: Math.max(position.y - AXIS_LABEL_LIFT, PLOT_TOP) - AXIS_LABEL_HEIGHT,
    wide: LABEL * 2,
    high: AXIS_LABEL_HEIGHT,
  }
}

/** Returns the CSS position style for an axis number. */
export const getAxisNumber = (y: number, offsetY: string, high = PLOT_HEIGHT): CSSProperties => ({
  insetInlineStart: `${(PLOT_LEFT / PLOT_WIDTH) * 100}%`,
  insetBlockStart: `${(y / high) * 100}%`,
  translate: `0 ${offsetY}`,
})

export const getAxisNumberBox = (y: number, offsetY: string): Box => ({
  x: PLOT_LEFT,
  y: offsetY === '0' ? y : y - AXIS_LABEL_HEIGHT,
  wide: AXIS_LABEL_WIDTH,
  high: AXIS_LABEL_HEIGHT,
})
