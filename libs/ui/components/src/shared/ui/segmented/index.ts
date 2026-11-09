export { default as SegmentedControl } from './SegmentedControl.vue'

/** One of the choices a segmented control offers. */
export interface SegmentedChoice {
  readonly id: string
  readonly text: string
}
