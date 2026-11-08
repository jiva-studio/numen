export { default as Select } from './Select.vue'
export { default as MultiSelect } from './MultiSelect.vue'

/** One of the choices a select offers. */
export interface SelectChoice {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  readonly isDisabled?: boolean
}
