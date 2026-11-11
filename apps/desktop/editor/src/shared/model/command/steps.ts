/**
 * A list a palette step offers, in groups of rows. A feature that builds such a
 * list and the palette that draws it both take the shape from here.
 */

/** One of a list the window itself holds, as the step that offers it draws it. */
export interface StepRow {
  readonly id: string
  readonly title: string
  readonly detail?: string
  readonly isDisabled?: boolean
  readonly isCurrent?: boolean
}

/** One group of such a list, named by whatever holds it. */
export interface StepGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly StepRow[]
  readonly silence?: string
}
