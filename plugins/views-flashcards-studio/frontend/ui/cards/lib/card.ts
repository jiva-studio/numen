/**
 * What a deck and a card model are both written in: the slots a card model names,
 */

export interface FieldValue {
  readonly field: string
  readonly text: string
}

/** One card model a card may be cut by: the word it is shown as, and its slots. */
export interface CardModel {
  readonly name: string
  /** The slots it names, in the order a person is asked for them. */
  readonly fields: readonly string[]
}
