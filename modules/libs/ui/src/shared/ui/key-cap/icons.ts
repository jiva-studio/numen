/**
 * What draws each key a cap holds, and how large.
 *
 * The icons are Lucide's, which is the set `components.json` names. Each is
 * drawn on a 24 grid and stroked at a width given here, so a cap is one line of
 * ink: every icon weighs what the letter beside it weighs.
 */
import {
  ArrowBigUp,
  ChevronUp,
  Command,
  CornerDownLeft,
  Option,
  type LucideIcon,
} from '@lucide/vue'
import type { PaletteIcon } from './keys'

/**
 * How thick an icon is stroked, in the 24 units it is drawn on, for an icon
 * filling the cap's icon box exactly. It is the width of the letter's stem in
 * the sans face this is designed against, at the type a cap is set in.
 */
const STROKE = 2.4

/** One key held, as it is drawn. */
export interface DrawnIcon {
  readonly icon: LucideIcon
  /**
   * How many icon boxes across this one is drawn. An icon drawn larger is
   * stroked thinner by as much, so its ink is the same thickness.
   */
  readonly fills: number
  /** How thick to stroke it, in the 24 units it is drawn on. */
  readonly stroke: number
  /** What it is called, for a reader who is listening rather than looking. */
  readonly said: string
}

const createIcon = (icon: LucideIcon, said: string, fills = 1): DrawnIcon => ({
  icon,
  fills,
  stroke: STROKE / fills,
  said,
})

/**
 * Every key a cap can hold. Four are Lucide's own key icons and fill the box.
 *
 * Control is a chevron, which Lucide draws twelve units wide where it draws the
 * shift arrow's head sixteen. It is given the four thirds that makes the two
 * the same width, and the thinner stroke that keeps its ink the same weight.
 */
export const ICONS = {
  control: createIcon(ChevronUp, 'Control', 4 / 3),
  shift: createIcon(ArrowBigUp, 'Shift'),
  command: createIcon(Command, 'Command'),
  option: createIcon(Option, 'Option'),
  return: createIcon(CornerDownLeft, 'Return'),
} as const satisfies Record<PaletteIcon, DrawnIcon>
