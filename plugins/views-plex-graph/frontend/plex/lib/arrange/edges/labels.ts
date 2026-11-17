import {
  getHeading,
  type EdgeCurve,
  type EdgeHeading,
  type EdgeSide,
  type EdgeTitle,
  type LabelType,
} from '../../edge'
import type { EdgeLabel } from '../../../../types'
import { getDepth, LABEL_GAP } from './ink'
import { MIDDLE, getLabelPositions } from './positions'

/**
 * How far from the line the center of a title is: nothing for a title on the line, whose
 * lowercase letters are centered on it; else half of the depth of the ink and the gap, so the
 * near edge of the letters is `LABEL_GAP` from the line.
 */
export const getShift = (side: EdgeSide, type: LabelType): number =>
  side * (getDepth(type) / 2 + LABEL_GAP)

/**
 * How far below the center of a title its baseline is, across the line. The center of a title
 * on the line is the middle of the x-height band, of a title beside it the middle of the ink
 * from capital to descender. The baseline is set by this alone, so no engine's own
 * idea of a middle decides where the letters stand.
 */
export const getRise = (side: EdgeSide, type: LabelType): number =>
  side === 0 ? type.xHeight / 2 : (type.ascent - type.descent) / 2

/**
 * Where and how each label of one line is set, in the order the labels are written.
 * One label is centered along the line and on it, its vertical middle on the line, read the readable way along it.
 * Two labels are set near the two ends, on opposite sides. When both records have a head, each label is near the end its head is at and reads toward it;
 * otherwise each is near the note that wrote it and reads away from it, the first on the left of the line taken from `from`, the second on the right.
 */
export function getLabelPlaces(labels: readonly EdgeLabel[]): LabelPlace[] {
  if (labels.length <= 1) return labels.map(() => ({ position: MIDDLE, side: 0 }))
  const positions = getLabelPositions(labels.length)
  const isHeaded = labels.every((label) => label.end !== undefined)
  return labels.map((label, index) => {
    const isToward = isHeaded ? label.end === 'to' : index === 0
    const position = isHeaded ? (isToward ? positions.at(-1)! : positions[0]!) : positions[index]!
    const side: EdgeSide = isToward ? -1 : 1
    return { position, side, heading: isToward ? 'along' : 'against' }
  })
}

/** Where a label is set: its place along the line, its side, and the way it is read where its record fixes that. */
interface LabelPlace {
  readonly position: number
  readonly side: EdgeSide
  readonly heading?: EdgeHeading
}

/**
 * The title set at a fraction of the curve from its `from` end, measured along
 * the curve the way its words are read.
 */
export function createTitle(
  curve: EdgeCurve,
  ...[position, words, title]: [position: number, words: string, title: TitleSetting]
): EdgeTitle {
  const { anchor = position, side, shift, rise } = title
  const heading = title.heading ?? getHeading(curve, position)
  return {
    anchor,
    heading,
    side,
    shift,
    rise,
    words,
    wordsAt: heading === 'against' ? 1 - position : position,
  }
}

/** What a title keeps wherever it is moved to along its line. */
export interface TitleSetting {
  readonly anchor?: number
  readonly heading?: EdgeHeading
  readonly side: EdgeSide
  readonly shift: number
  readonly rise: number
}
