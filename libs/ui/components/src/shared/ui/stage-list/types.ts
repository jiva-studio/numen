export type StageState = 'done' | 'active' | 'pending'

export interface Stage {
  id: string
  label: string
  state: StageState
  detail?: string
}
