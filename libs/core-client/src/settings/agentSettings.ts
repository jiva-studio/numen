/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType =
  'boolean' | 'string' | 'number' | 'enum' | 'range' | 'file' | 'folder'

/** Where the active agent program and default AI agent stand in the settings file. */
export const AGENT_USE_PATH = ['agent', 'use'] as const
export const DEFAULT_AGENT_PATH = ['ai', 'default', 'agent'] as const
export const DEFAULT_MODEL_PATH = ['ai', 'default', 'model'] as const

export const getProgramModelPath = (program: string): readonly string[] => [
  'agent',
  program,
  'model',
]
export const getProgramStepsPath = (program: string): readonly string[] => [
  'agent',
  program,
  'max_steps',
]
export const getProgramHooksPath = (program: string): readonly string[] => [
  'agent',
  program,
  'reads_hooks_and_skills',
]

/** The agent program and the model a setting of that type holds. Empty is the default. */
export interface CompletionChoice {
  readonly agent: string
  readonly model: string
}

/** Reads the agent and model choice from a raw setting value. */
export const readAgentChoice = (value: unknown): CompletionChoice => {
  if (typeof value !== 'object' || value === null) return { agent: '', model: '' }
  const fields = value as Record<string, unknown>
  return {
    agent: typeof fields['agent'] === 'string' ? fields['agent'] : '',
    model: typeof fields['model'] === 'string' ? fields['model'] : '',
  }
}
