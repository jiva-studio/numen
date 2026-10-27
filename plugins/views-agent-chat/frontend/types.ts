/**
 * Types and interfaces for agent conversation tabs.
 */
import type { AgentModel, Span } from '@numen/plugin-sdk'

/** External dependencies required by an agent tab. */
export interface AgentTabDeps {
  openFileAt(path: string, ...spans: readonly Span[]): void
  openFileBeside(path: string): void
  resolveLinks(text: string): Promise<ReadonlyMap<string, string>>
  unreachable(): string
  writeSetting?(path: readonly string[], value: unknown): Promise<void>
  getSetting?(path: readonly string[]): unknown
  getModelsAt?(path: readonly string[]): readonly AgentModel[]
}
