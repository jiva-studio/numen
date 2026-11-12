/**
 * Types and interfaces for agent conversation tabs.
 */
import type { TextLink, TextRange } from '@numen/plugin-sdk'
import type { LinkPiece } from '@numen/ui'

/** The voice of a turn of a conversation. */
export type Voice = 'prompt' | 'response' | 'doing' | 'plan'

/** How far along a turn is. Defaults to settled. */
export type TurnState = 'settled' | 'arriving' | 'failed'

/** How far the agent is with one entry of its plan. */
export type PlanStatus = 'pending' | 'active' | 'done'

/** One entry of the plan of the agent. */
export interface PlanItem {
  readonly text: string
  readonly status: PlanStatus
}

/**
 * A turn with offersSettings is a failed turn that says no agent is set up, under which the settings are
 * offered.
 */
export interface Turn {
  readonly id: string
  readonly voice: Voice
  readonly text: string
  readonly subject?: string
  readonly state?: TurnState
  readonly canOpen?: boolean
  readonly offersSettings?: boolean
  readonly plan?: readonly PlanItem[]
  readonly unresolved?: readonly string[]
  readonly links?: readonly LinkPiece[]
}

/** External dependencies required by an agent tab. */
export interface AgentTabDeps {
  openFile(path: string): void
  openFileAt(path: string, ...spans: readonly TextRange[]): void
  openFileAdjacent(path: string): void
  resolveLinks(text: string): Promise<readonly TextLink[]>
  /** Opens the settings page of the plugin, where its agents are chosen. */
  openSettings(): void
}
