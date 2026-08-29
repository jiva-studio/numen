/**
 * What draws each key a cap holds, and how large.
 *
 * The marks are Lucide's, which is the set `components.json` names. Each is
 * drawn on a 24 grid and stroked at a width given here, so a cap is one line of
 * ink: every mark weighs what the letter beside it weighs.
 */
import {
  ArrowBigUp,
  ChevronUp,
  Command,
  CornerDownLeft,
  Option,
  type LucideIcon,
} from '@lucide/vue'
import type { PaletteMark } from './model'

/**
 * How thick a mark is stroked, in the 24 units it is drawn on, for a mark
 * filling the cap's mark box exactly.
 *
 * In the sans face this is designed against, the letter's stem is 0.086em at
 * the type a cap is set in and its capitals are 0.713em tall; a box of 0.85em
 * carries a 20-unit mark at that height, and 2.4 units of stroke land on that
 * stem. A face that draws its stems differently moves what the marks sit
 * beside, and none of it moves the marks.
 */
const STROKE = 2.4

/** One key held, as it is drawn. */
export interface DrawnMark {
  readonly icon: LucideIcon
  /**
   * How many mark boxes across this one is drawn. A mark drawn larger is
   * stroked thinner by as much, so its ink is the same thickness.
   */
  readonly fills: number
  /** How thick to stroke it, in the 24 units it is drawn on. */
  readonly stroke: number
  /** What it is called, for a reader who is listening rather than looking. */
  readonly said: string
}

const drawn = (icon: LucideIcon, said: string, fills = 1): DrawnMark => ({
  icon,
  fills,
  stroke: STROKE / fills,
  said,
})

/**
 * Every key a cap can hold. Four are Lucide's own key marks and fill the box.
 *
 * Control is a chevron, which Lucide draws twelve units wide where it draws the
 * shift arrow's head sixteen. It is given the four thirds that makes the two
 * the same width, and the thinner stroke that keeps its ink the same weight.
 */
export const MARKS = {
  control: drawn(ChevronUp, 'Control', 4 / 3),
  shift: drawn(ArrowBigUp, 'Shift'),
  command: drawn(Command, 'Command'),
  option: drawn(Option, 'Option'),
  return: drawn(CornerDownLeft, 'Return'),
} as const satisfies Record<PaletteMark, DrawnMark>
