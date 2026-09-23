export { default as Select } from './Select.vue'

/** One of the choices a select offers. */
export interface SelectChoice {
  /** The caller's own identifier, handed back as given. */
  readonly id: string
  /** What is drawn on the row. */
  readonly text: string
  /** A second line under the words, in the small print. */
  readonly detail?: string
  /** The shelf it stands under. Choices naming one shelf are drawn together. */
  readonly group?: string
}
