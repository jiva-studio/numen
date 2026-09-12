/**
 * A list as it is drawn: rows carrying a title and a line under it, gathered
 * into named groups.
 */

/** One of a list the window itself holds, as the step that offers it draws it. */
export interface StepRow {
  readonly id: string
  readonly title: string
  /** A second line: what is true of this row and not of the ones beside it. */
  readonly detail?: string
  /** Drawn, said, and not chosen. */
  readonly disabled?: boolean
  /** The value the setting this list is of holds now, which is where it opens. */
  readonly inForce?: boolean
}

/** One group of such a list, named by whatever holds it. */
export interface StepGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly StepRow[]
  /** What is said in its place where it holds nothing. */
  readonly silence?: string
}
