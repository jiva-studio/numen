import type { Position } from '@numen/ui'
import {
  BACKLOG_PLOT,
  PLOT_BOTTOM,
  PLOT_HEIGHT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_TOP,
  PLOT_WIDTH,
  type Extent,
} from './plot'
import { CALLOUT_GAP, CALLOUT_HEIGHT, CALLOUT_WIDTH, LABEL } from './labelSizes'
import {
  areBoxesSeparate,
  getAxisNumber,
  getAxisNumberBox,
  getMarkNameStyle,
  getNameBox,
  isAxisLabelClear,
  type Box,
  type InlinePositionStyle,
} from './labelBox'

type CSSProperties = InlinePositionStyle

/** Position and bounding box of the knob callout. */
export interface Callout {
  readonly isUnder: boolean
  readonly box: Box
  readonly at: CSSProperties
  readonly tail: CSSProperties
}

/** Creates the callout position and bounding box for the knob. */
export const createCallout = (knob: Position): Callout => {
  const isUnder = knob.y - CALLOUT_GAP - CALLOUT_HEIGHT < PLOT_TOP
  const half = CALLOUT_WIDTH / 2
  let back = half
  if (knob.x < PLOT_LEFT + half) back = 0
  else if (knob.x > PLOT_RIGHT - half) back = CALLOUT_WIDTH
  const edge = isUnder ? knob.y + CALLOUT_GAP : knob.y - CALLOUT_GAP
  return {
    isUnder,
    box: {
      x: knob.x - back,
      y: isUnder ? edge : edge - CALLOUT_HEIGHT,
      wide: CALLOUT_WIDTH,
      high: CALLOUT_HEIGHT,
    },
    at: {
      insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`,
      insetBlockStart: `${(edge / PLOT_HEIGHT) * 100}%`,
      translate: `${(-back / CALLOUT_WIDTH) * 100}% ${isUnder ? '0' : '-100%'}`,
    },
    tail: {
      insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`,
      insetBlockStart: `${(edge / PLOT_HEIGHT) * 100}%`,
    },
  }
}

/** Mark on the plot with its label text. */
export interface Mark {
  readonly key: string
  readonly at: Position
  readonly text: string
}

/** Positioned mark label with its bounding box. */
export interface Label {
  readonly key: string
  readonly text: string
  readonly at: CSSProperties
  readonly box: Box
}

/** Creates non-overlapping labels for the marks. */
export const createLabels = (marks: readonly Mark[], calloutBox: Box | null): readonly Label[] => {
  const occupiedBoxes: Box[] = calloutBox ? [calloutBox] : []
  const out: Label[] = []
  for (const mark of marks) {
    if (!mark.text) continue
    const box = getNameBox(mark.at)
    if (!occupiedBoxes.every((one) => areBoxesSeparate(box, one))) continue
    occupiedBoxes.push(box)
    out.push({ key: mark.key, text: mark.text, at: getMarkNameStyle(mark.at), box })
  }
  return out
}

/** Axis height label with its position and bounding box. */
export interface Height {
  readonly at: CSSProperties
  readonly box: Box
  readonly text: string
}

/** Context for placing axis height labels. */
export interface AxisLabelsContext {
  places: readonly Position[]
  marks: readonly Position[]
  calloutBox?: Box | null
}

export const createAxisLabels = (
  extent: Extent,
  context: AxisLabelsContext,
  getText: (value: number) => string,
): readonly Height[] => {
  const { min: least, max: most } = extent
  const { places, marks, calloutBox = null } = context
  const measureHeightAt = (y: number, offsetY: string, value: number): readonly Height[] => {
    const box = getAxisNumberBox(y, offsetY)
    if (!isAxisLabelClear(y, places, marks)) return []
    if (calloutBox && !areBoxesSeparate(box, calloutBox)) return []
    return [{ at: getAxisNumber(y, offsetY), box, text: getText(value) }]
  }
  if (most === least) {
    return [
      {
        at: getAxisNumber(PLOT_BOTTOM, '0'),
        box: getAxisNumberBox(PLOT_BOTTOM, '0'),
        text: getText(most),
      },
    ]
  }
  return [...measureHeightAt(PLOT_TOP, '-100%', most), ...measureHeightAt(PLOT_BOTTOM, '0', least)]
}

/** Creates the axis height labels for the backlog plot. */
export const createBacklogAxisLabels = (
  extent: Extent,
  places: readonly Position[],
  formatValue: (value: number) => string,
): readonly { at: CSSProperties; text: string }[] => {
  const { min: least, max: most } = extent
  const createLabel = (y: number, offsetY: string, value: number) =>
    isAxisLabelClear(y, places, [])
      ? [{ at: getAxisNumber(y, offsetY, BACKLOG_PLOT.high), text: formatValue(value) }]
      : []
  if (most === least) {
    return [
      { at: getAxisNumber(BACKLOG_PLOT.foot, '0', BACKLOG_PLOT.high), text: formatValue(least) },
    ]
  }
  return [
    ...createLabel(BACKLOG_PLOT.top, '-100%', most),
    ...createLabel(BACKLOG_PLOT.foot, '0', least),
  ]
}

export const getKnobValueStyle = (knob: Position | null): CSSProperties => {
  if (!knob) return {}
  let back = '-50%'
  if (knob.x < PLOT_LEFT + LABEL) back = '0'
  else if (knob.x > PLOT_RIGHT - LABEL) back = '-100%'
  return { insetInlineStart: `${(knob.x / PLOT_WIDTH) * 100}%`, translate: `${back} 0` }
}
