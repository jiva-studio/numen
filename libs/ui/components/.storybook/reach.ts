/**
 * What the keyboard needs wherever it stops, and what a stop is described
 * by. The walk that finds the stops is `keyboard.ts`, and needs a browser;
 * this half decides, and decides from four words.
 */

/** One place the keyboard stopped. */
export interface Stop {
  readonly where: string
  readonly name: string
  readonly shown: boolean
  readonly isMoving: boolean
}

/** What a story turned out to be, walked from the top of its tab order. */
export interface Walk {
  readonly stops: readonly Stop[]
  readonly trapped: string | null
  readonly settled: boolean
}

/**
 * Every place the keyboard stops is drawn where a person can see it, says what
 * it is, and hands Tab on. A stop the page says is still arriving is held at
 * the opacity it was passing through and so is not asked whether it can be seen.
 *
 * `keeps` is a story saying Tab stays where it is, and how a person leaves.
 */
export function faults({ stops, trapped }: Walk, keeps = false): string[] {
  const wrong: string[] = []
  if (trapped && !keeps) wrong.push(`${trapped} answers Tab by keeping it`)
  for (const stop of stops) {
    const at = `${stop.where}${stop.name ? ` (${stop.name})` : ''}`
    if (!stop.shown && !stop.isMoving) wrong.push(`${at} is a stop a person cannot see`)
    if (!stop.name) wrong.push(`${stop.where} is a stop with no name to read out`)
  }
  return wrong
}
