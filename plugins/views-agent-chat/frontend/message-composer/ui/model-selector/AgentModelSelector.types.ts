import type { AgentModelOption } from '../../model/options'
import type { Timer } from '../../model/opening'

/** pendingAgentIds are the agents whose models are not listed yet: the list waits for them before it opens. */
export interface AgentModelSelectorProps {
  options?: readonly AgentModelOption[]
  selectedAgentId?: string
  selectedModelId?: string
  isDisabled?: boolean
  emptyLabel?: string
  pendingAgentIds?: readonly string[]
  loadingLabel?: string
  timer?: Timer
}
