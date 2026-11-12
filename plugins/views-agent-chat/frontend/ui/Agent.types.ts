import type { Turn } from '../types'

export interface AgentProps {
  turns: readonly Turn[]
  isAnswerPending?: boolean
  placeholder?: string
  isDisabled?: boolean
  sendLabel?: string
  stopLabel?: string
}
