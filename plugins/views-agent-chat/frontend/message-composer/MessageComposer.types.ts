import type { AgentModelOption } from './model-selector'
import type { ReasoningLevelOption } from './reasoning-selector'

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
}
