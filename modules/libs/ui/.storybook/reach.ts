/**
 * What the keyboard is owed wherever it stops, and what a stop is described
 * by. The walk that finds the stops is `keyboard.ts`, and needs a browser;
 * this half decides, and decides from four words.
 */

/** One place the keyboard stopped. */
export interface Stop {
  /** Where it stands, said so a person can find it in the source. */
  readonly where: string
  /** What a screen reader would call it. */
  readonly name: string
  /** Whether anything is painted differently once the keyboard is on it. */
  readonly rings: boolean
  /** Whether the browser draws it where a person could see it. */
  readonly shown: boolean
  /** Whether the page says it is still on its way in or out. */
  readonly moving: boolean
  /** Whether it is typed into, and so draws a caret of its own. */
  readonly typed: boolean
}

/** What a story turned out to be, walked from the top of its tab order. */
export interface Walk {
  readonly stops: readonly Stop[]
  /** Where Tab went and stayed, having kept it through every strike it was given. */
  readonly trapped: string | null
  /** Whether the page had come to rest before the census was taken. */
  readonly settled: boolean
}

/**
 * Every place the keyboard stops is drawn where a person can see it, says what
 * it is, shows that the keyboard is there, and hands Tab on. A thing typed into
 * draws its own caret, and a stop the page says is still arriving is held at
 * the opacity it was passing through and so is not asked whether it can be seen.
 *
 * `keeps` is a story saying Tab stays where it is, and how a person leaves.
 */
export function faults({ stops, trapped }: Walk, keeps = false): string[] {
  const wrong: string[] = []
  if (trapped && !keeps) wrong.push(`${trapped} answers Tab by keeping it`)
  for (const stop of stops) {
    const at = `${stop.where}${stop.name ? ` (${stop.name})` : ''}`
    if (!stop.shown && !stop.moving) wrong.push(`${at} is a stop a person cannot see`)
    if (!stop.name) wrong.push(`${stop.where} is a stop with no name to read out`)
    if (!stop.rings && !stop.typed) wrong.push(`${at} draws nothing when the keyboard lands on it`)
  }
  return wrong
}
