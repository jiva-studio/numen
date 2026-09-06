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
  /** Where Tab went and stayed, having refused to move twice running. */
  readonly trapped: string | null
  /** Whether the page had come to rest before the census was taken. */
  readonly settled: boolean
}

/**
 * Every place the keyboard stops is drawn where a person can see it, says what
 * it is, and shows that the keyboard is there.
 *
 * A thing typed into is not asked for a ring: its caret is where the keyboard
 * is, and a second mark around a field a person is typing in is noise.
 *
 * A stop the page says is still on its way is not asked whether it can be
 * seen. The walk holds the page still to read a colour off it, and something
 * held still halfway in is held at the opacity it was passing through: what
 * that reading answers is the holding, not the page. It is asked everything
 * else, and a stop standing at rest and drawn nowhere is a fault as it was.
 * Nothing else is excused.
 *
 * What answers Tab by keeping it is not judged here. Two things in this
 * library do it — a palette a person leaves with Escape, and the editor, which
 * indents until Escape hands Tab back. Each is left with the keyboard alone,
 * and a rule that could tell a way out from none would need a list.
 *
 * A native time field is not one of them. It spends Tab on its own hours and
 * minutes and hands it on after the last of them, in both engines, so the walk
 * waits it out instead.
 */
export function faults({ stops }: Walk): string[] {
  const wrong: string[] = []
  for (const stop of stops) {
    const at = `${stop.where}${stop.name ? ` (${stop.name})` : ''}`
    if (!stop.shown && !stop.moving) wrong.push(`${at} is a stop a person cannot see`)
    if (!stop.name) wrong.push(`${stop.where} is a stop with no name to read out`)
    if (!stop.rings && !stop.typed) wrong.push(`${at} draws nothing when the keyboard lands on it`)
  }
  return wrong
}
