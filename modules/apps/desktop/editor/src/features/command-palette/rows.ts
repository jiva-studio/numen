/**
 * The lists a step of the palette stands on, and what the keyboard standing on
 * one means.
 *
 * A list is the window's own: the themes it can be dressed in, the sizes it can
 * be drawn at, the two answers to a setting. The palette draws whatever it is
 * handed, and whoever holds the list says what a row is and what choosing it
 * does.
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

/**
 * The lists the window holds, and what it does with the one the keyboard is
 * standing on. A list is read again every time the step is drawn, so what the
 * window holds may change while the step stands open.
 */
export interface PaletteLists {
  /**
   * What this command offers now, in the groups it is drawn in. The words typed
   * come too: a list may hold a row made out of them.
   */
  getStepGroups(command: string, typed: string): readonly StepGroup[]
  /** The one the keyboard is standing on, and nothing where it stands on none. */
  previewItem(command: string, item: string): void
}

/**
 * What the window knows about a note by the name it is filed under. A step
 * stands open while the vault moves under it, and this is read again each time
 * the step is drawn and once more as the invocation is made.
 */
export interface NoteLookup {
  /** What it is called now, and nothing where the window names it nothing. */
  getTitle(path: string): string
  /** The identity of the tab holding it, and nothing where none holds it. */
  holding(path: string): string | null
}
