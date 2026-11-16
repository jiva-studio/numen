import type { LabelType } from '../../edge'

/** The depth of one line of a title across its line: from the top of a capital to the bottom of a descender. */
export const getDepth = (type: LabelType): number => type.ascent + type.descent

/**
 * The gap, in plex units, between the line and the near edge of the letters of
 * a label of a pair: as close as the letters go without touching the line.
 * It is the same for every kind of line and for both labels of a pair.
 */
export const LABEL_GAP = 2
