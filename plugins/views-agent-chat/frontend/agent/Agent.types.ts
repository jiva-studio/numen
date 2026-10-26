import type { AgentModelOption, ReasoningLevelOption } from '../message-composer'
import type { Turn } from '../thread'

export interface AgentProps {
  turns: readonly Turn[]
  /** An answer is on its way. */
  isWorking?: boolean
  placeholder?: string
  isDisabled?: boolean
  /** What the disc at the end of the field is called while it sends. */
  sendLabel?: string
  /** What it is called while it stops the answer on its way. */
  stopLabel?: string
  options?: readonly AgentModelOption[]
  selectedAgentId?: string
  selectedModelId?: string
  reasoningLevels?: readonly ReasoningLevelOption[]
  selectedReasoningId?: string
  reasoningName?: string
}
