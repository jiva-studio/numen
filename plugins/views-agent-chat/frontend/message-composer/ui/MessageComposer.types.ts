import type { AgentModelOption, ReasoningLevelOption } from '../model/options'

export interface MessageComposerProps {
  placeholder?: string
  isAnswerPending?: boolean
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
