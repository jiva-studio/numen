export type StageState = 'done' | 'active' | 'pending'

export interface Stage {
  id: string
  label: string
  state: StageState
  /** What the stage found, kept at the end of its row. */
  detail?: string
}
