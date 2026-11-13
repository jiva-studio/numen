import type { AgentModelOption } from '../../model/options'
import type { Timer } from '../../model/opening'

export interface AgentModelSelectorProps {
  options?: readonly AgentModelOption[]
  selectedAgentId?: string
  selectedModelId?: string
  isDisabled?: boolean
  emptyLabel?: string
  /** The agents whose models have not been listed yet; the list waits for them before it opens. */
  pendingAgentIds?: readonly string[]
  loadingLabel?: string
  timer?: Timer
}
