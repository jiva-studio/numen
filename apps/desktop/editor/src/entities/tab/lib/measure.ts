import type { NoteTabState } from '@numen/plugin-loader'

/** The state of a tab that lays its content out again on request. */
export type MeasurableState = Required<Pick<NoteTabState, 'measure'>>

/** Whether the state of a tab has the `measure` of the note tab contract. */
export const isMeasurable = (state: object): state is MeasurableState =>
  'measure' in state && typeof state.measure === 'function'
