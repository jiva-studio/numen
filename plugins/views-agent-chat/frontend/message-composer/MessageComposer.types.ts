import type { AgentModelOption } from './model-selector'
import type { ReasoningLevelOption } from './reasoning-selector'

export interface MessageComposerProps {
  placeholder?: string
  /** An answer is on its way; the disc stops it. */
  isAnswerPending?: boolean
  isDisabled?: boolean
  /** What the disc is called while it sends. */
  sendLabel?: string
  /** What it is called while it stops. */
  stopLabel?: string
  options?: readonly AgentModelOption[]
  selectedAgentId?: string
  selectedModelId?: string
  /** The depths of reasoning the chosen model offers. The selector shows only where there are some. */
  reasoningLevels?: readonly ReasoningLevelOption[]
  selectedReasoningId?: string
  /** What the reasoning selector is called. */
  reasoningName?: string
}
