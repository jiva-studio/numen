import type { SelectChoice } from '@numen/ui'
import type { CompletionChoice } from '@/entities/settings'

export interface AgentChoiceProps {
  /** What the two rows are about, which their names are addressed by. */
  at: string
  agentName: string
  agentDetail: string
  modelName: string
  modelDetail: string
  agents: readonly SelectChoice[]
  /** The models of the agent in force. None hides the row. */
  models: readonly SelectChoice[] | null
  choice: CompletionChoice
}
