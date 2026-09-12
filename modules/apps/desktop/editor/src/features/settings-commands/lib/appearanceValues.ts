/**
 * What the appearance commands are named by, and the two kinds of value a row
 * of theirs stands for.
 *
 * A row is reached by the name it is offered under, so the name has to say
 * which command it belongs to and what it names in it.
 */
import { MODES } from '@/entities/settings'
import type { Mode, Scales } from '@/entities/settings'

/** The command whose step offers the themes, and the one that offers the modes. */
export const APPEARANCE = 'appearance'
export const MODE = 'mode'

/** The command that offers how large the interface is drawn, and how large the text is set. */
export const INTERFACE_SCALE = 'interfaceScale'
export const TEXT_SCALE = 'textScale'

/** The commands whose step offers a list the appearance holds. */
export const APPEARANCE_COMMANDS: readonly string[] = [APPEARANCE, MODE, INTERFACE_SCALE, TEXT_SCALE]

/** Which of the two sizes a row is one of. */
export type ScaleKind = typeof INTERFACE_SCALE | typeof TEXT_SCALE

/** One size, and which of the two it is. */
export interface ScaleChoice {
  readonly which: ScaleKind
  readonly size: number
}

/**
 * How long the keyboard has to have stood on a size before the window is drawn
 * at it. A held arrow key crosses a row every 40 milliseconds, and each row
 * applied is every open document laid out again and its pages emptied.
 */
export const HELD = 150

/**
 * What a mode is offered under: the command it belongs to, and the mode. A
 * theme is named by its shelf, and there are two shelves, so no theme is ever
 * named this.
 */
export const getModeId = (one: Mode): string => `${MODE}:${one}`

/** The mode a row names, and nothing for a row naming a theme. */
export const modeOf = (item: string): Mode | null =>
  MODES.find((one) => getModeId(one) === item) ?? null

/**
 * What a size is offered under: the command it belongs to, and the multiplier.
 * A theme is named by its shelf, and there are two shelves, so no theme is
 * ever named this.
 */
export const getSizeId = (which: ScaleKind, size: number): string => `${which}:${size}`

/** The size a row names, and nothing for a row naming anything else. */
export const sizeOf = (item: string): ScaleChoice | null => {
  const [which, count] = item.split(':')
  if (which !== INTERFACE_SCALE && which !== TEXT_SCALE) return null
  const size = Number(count)
  return size > 0 ? { which, size } : null
}

/** What is said of one of the two sizes. */
export const its = <T,>(both: Scales<T>, which: ScaleKind): T => both[which]

/** The pair with what is said of one of the two put in its place. */
export const onto = <T,>(both: Scales<T>, which: ScaleKind, one: T): Scales<T> => ({
  ...both,
  [which]: one,
})
