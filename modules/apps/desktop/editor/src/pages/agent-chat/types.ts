/**
 * Types and interfaces for agent conversation tabs.
 */
import type { Span } from '@/shared/span'

/** External dependencies required by an agent tab. */
export interface AgentTabDeps {
  openFileAt(path: string, ...spans: readonly Span[]): void
  openFileBeside(path: string): void
  resolve(written: readonly string[]): Promise<ReadonlyMap<string, string>>
  unreachable(): string
  writeSetting?(path: readonly string[], value: unknown): Promise<void>
  getSetting?(path: readonly string[]): unknown
}

/** Reference to a note the conversation relates to. */
export interface NoteRef {
  readonly path: string
  readonly title: string
}
