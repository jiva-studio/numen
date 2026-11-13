import type { AgentModelOption, ReasoningLevelOption } from '../message-composer'
import type { Turn } from '../types'

export interface AgentProps {
  turns: readonly Turn[]
  isAnswerPending?: boolean
  placeholder?: string
  isDisabled?: boolean
  sendLabel?: string
  stopLabel?: string
  options?: readonly AgentModelOption[]
  selectedAgentId?: string
  selectedModelId?: string
  reasoningLevels?: readonly ReasoningLevelOption[]
  selectedReasoningId?: string
  reasoningName?: string
  isAgentMissing?: boolean
  noAgentLabel?: string
  noAgentMessage?: string
  openSettingsLabel?: string
  pendingAgentIds?: readonly string[]
  loadingLabel?: string
  failedAgentIds?: readonly string[]
  failedLabel?: string
}
