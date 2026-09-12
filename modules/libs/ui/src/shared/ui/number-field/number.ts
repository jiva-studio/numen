/**
 * What a line of typing comes to as a number, where the bounds leave it, and
 * where a key leaves it.
 *
 * Pure: the same text gives the same answer wherever it is read, so what the
 * field hands on can be named without drawing it.
 */

/** How far a number may go, and what an arrow key moves it by. */
export interface Bounds {
  readonly min: number
  readonly max: number
  readonly step: number
}

export const DEFAULT_BOUNDS: Bounds = { min: 0, max: 999, step: 1 }

/** Text a number could still be typed out of: a lone sign, a trailing point. */
const ON_ITS_WAY = /^[+-]?(\d+([.,]\d*)?|[.,]\d*)?$/

/** A comma between a whole number and a group of three, which is no point. */
const GROUPED = /^[+-]?[1-9]\d{0,2},\d{3}$/

/** Whether more typing could still make a number of the text. */
export const onItsWay = (typed: string): boolean => {
  const said = typed.trim()
  return ON_ITS_WAY.test(said) && !GROUPED.test(said)
}

/**
 * The number the text stands for; nothing where it stands for none. The shape
 * is what a person types: no exponent, no hexadecimal, no word for infinity.
 */
export const numberOf = (typed: string): number | null => {
  const said = typed.trim()
  if (said === '' || !onItsWay(said)) return null
  const value = Number(said.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

/** The number brought inside the bounds. */
export const clamp = (value: number, bounds: Bounds): number =>
  Math.min(bounds.max, Math.max(bounds.min, value))

/** The places a step is written to, which is what a number moved by it is kept to. */
const places = (step: number): number => {
  const said = `${step}`
  const point = said.indexOf('.')
  return point < 0 ? 0 : said.length - point - 1
}

/** The place the step lays nearest a number, counted from the floor. */
const onStep = (value: number, bounds: Bounds): number => {
  if (bounds.step <= 0) return value
  const steps = Math.round((value - bounds.min) / bounds.step)
  const scale = 10 ** places(bounds.step)
  return Math.round((bounds.min + steps * bounds.step) * scale) / scale
}

/** Where the bounds leave a number: on a place the step lays, inside the ends. */
export const snapToBounds = (value: number, bounds: Bounds): number =>
  clamp(onStep(value, bounds), bounds)

/** Whether the text is a number the bounds allow. An empty field is neither. */
export const isAllowed = (typed: string, bounds: Bounds): boolean => {
  const value = numberOf(typed)
  return value !== null && value === snapToBounds(value, bounds)
}

/** Where an arrow key leaves the number: one step from where it stands. */
export const stepBy = (value: number | null, by: number, bounds: Bounds): number =>
  snapToBounds((value ?? bounds.min) + by * bounds.step, bounds)

/** How many steps a page key covers at once. */
const PACES = 10

/** Where a key leaves the number, and nothing for a key the field does not answer. */
export const stepForKey = (key: string, value: number | null, bounds: Bounds): number | null => {
  if (key === 'Home') return snapToBounds(bounds.min, bounds)
  if (key === 'End') return snapToBounds(bounds.max, bounds)
  if (key === 'ArrowUp') return stepBy(value, 1, bounds)
  if (key === 'ArrowDown') return stepBy(value, -1, bounds)
  if (key === 'PageUp') return stepBy(value, PACES, bounds)
  if (key === 'PageDown') return stepBy(value, -PACES, bounds)
  return null
}

/** How a number is written into the field. */
export const written = (value: number | null): string => (value === null ? '' : String(value))

/** Whether what is typed stands for the number in force. An empty field holds none. */
export const isTextForValue = (typed: string, value: number | null): boolean =>
  value === null ? typed.trim() === '' : numberOf(typed) === value
