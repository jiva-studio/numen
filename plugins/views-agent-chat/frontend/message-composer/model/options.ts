/** What a person picks from in the composer, as plain values. */

export interface ReasoningLevelOption {
  readonly id: string
  readonly title: string
}

export interface AgentModelOption {
  readonly agentId: string
  readonly agentTitle: string
  readonly modelId: string
  readonly modelTitle: string
  readonly isAvailable: boolean
  readonly isDefault?: boolean
  readonly reasoningLevels?: readonly ReasoningLevelOption[]
  readonly defaultReasoning?: string
}
