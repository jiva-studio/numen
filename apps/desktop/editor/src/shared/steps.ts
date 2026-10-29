/**
 * A list a palette step offers, in groups of rows. A feature that builds such a
 * list and the palette that draws it both take the shape from here.
 */

/** One of a list the window itself holds, as the step that offers it draws it. */
export interface StepRow {
  readonly id: string
  readonly title: string
  /** A second line: what applies to this row and not to the others. */
  readonly detail?: string
  /** Shown but not selectable. */
  readonly isDisabled?: boolean
  /** Whether the row holds the setting's current value, which is where the list opens. */
  readonly isCurrent?: boolean
}

/** One group of such a list, named by whatever holds it. */
export interface StepGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly StepRow[]
  /** The text shown in place of the items when the group is empty. */
  readonly silence?: string
}
