import type { Unvalidated } from '../errors'

/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType =
  'boolean' | 'string' | 'number' | 'enum' | 'range' | 'file' | 'folder'

/** Where the active agent program and default AI agent stand in the settings file. */
export const AGENT_USE_PATH = ['app', 'agent', 'use'] as const
export const DEFAULT_AGENT_PATH = ['app', 'ai', 'default', 'agent'] as const
export const DEFAULT_MODEL_PATH = ['app', 'ai', 'default', 'model'] as const

export const getProgramModelPath = (program: string): readonly string[] => [
  'app',
  'agent',
  program,
  'model',
]

/** The agent program and the model a setting of that type holds. Empty is the default. */
export interface CompletionChoice {
  readonly agent: string
  readonly model: string
}

/** Reads the agent and model choice from a raw setting value. */
export const parseAgentChoice = (value: Unvalidated): CompletionChoice => {
  if (typeof value !== 'object' || value === null) return { agent: '', model: '' }
  return {
    agent: 'agent' in value && typeof value.agent === 'string' ? value.agent : '',
    model: 'model' in value && typeof value.model === 'string' ? value.model : '',
  }
}
