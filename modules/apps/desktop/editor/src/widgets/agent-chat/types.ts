/**
 * Types and interfaces for agent conversation tabs.
 */
import type { Span } from '@/shared/span'

/** External dependencies required by an agent tab. */
export interface AgentTabDeps {
  opens(path: string, ...spans: readonly Span[]): void
  beside(path: string): void
  resolve(written: readonly string[]): Promise<ReadonlyMap<string, string>>
  unreachable(): string
}

/** Reference to a note the conversation relates to. */
export interface NoteRef {
  readonly path: string
  readonly title: string
}
