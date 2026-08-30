export { default as Segmented } from './Segmented.vue'

/** One of the choices a segmented control offers. */
export interface SegmentedChoice {
  /** The caller's own identifier, handed back as given. */
  readonly id: string
  /** What is drawn on the segment. */
  readonly text: string
}
