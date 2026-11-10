/**
 * Types and interfaces for agent conversation tabs.
 */
import type { AgentModel, JsonValue, TextLink, TextRange } from '@numen/plugin-sdk'

/** External dependencies required by an agent tab. */
export interface AgentTabDeps {
  openFileAt(path: string, ...spans: readonly TextRange[]): void
  openFileAdjacent(path: string): void
  resolveLinks(text: string): Promise<readonly TextLink[]>
  unreachable(): string
  settings?: AgentSettingsDeps
}

/** The settings of the window an agent tab reads and writes; a tab without them offers no models. */
export interface AgentSettingsDeps {
  writeSetting?(path: readonly string[], value: JsonValue): Promise<void>
  getSetting?(path: readonly string[]): JsonValue | undefined
  /** The depth of reasoning the plugin setting `reasoning` keeps; empty where it keeps none. */
  getReasoning?(): string
  getModelsAt?(path: readonly string[]): readonly AgentModel[]
  /** Opens the settings page where the agents are configured. */
  openSettings?(): void
}
