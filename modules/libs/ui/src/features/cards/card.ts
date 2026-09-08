/**
 * What a deck and a stencil are both written in: the slots a stencil names,
 * and what a card stands in them.
 */

/** One named slot and what stands in it. */
export interface FieldValue {
  readonly field: string
  readonly text: string
}

/** One stencil a card may be cut by: the word it is shown as, and its slots. */
export interface Stencil {
  readonly name: string
  /** The slots it names, in the order a person is asked for them. */
  readonly fields: readonly string[]
}
