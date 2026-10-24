/**
 * Resolves action links for failed background tasks.
 */
import type { Task } from '@/shared/notices/task'

export interface PluginMeta {
  readonly id: string
  readonly name: string
}

const NON_CONFIGURABLE_KEYWORDS: readonly string[] = [
  'no such file',
  'not found',
  'does not exist',
  'is a directory',
  'permission denied',
  'stat ',
  'read ',
]

const CONFIGURABLE_KEYWORDS: readonly string[] = [
  'authenticate',
  'oauth',
  'session expired',
  'credentials',
  'api key',
  'not configured',
  'unconfigured',
  'unauthorized',
  'provider',
  'no model',
  'quota',
  'rate limit',
]

export function isConfigurableError(error: string): boolean {
  if (!error) return false
  const err = error.toLowerCase()
  if (NON_CONFIGURABLE_KEYWORDS.some((kw) => err.includes(kw))) {
    return false
  }
  return CONFIGURABLE_KEYWORDS.some((kw) => err.includes(kw))
}

const AGENT_KEYWORDS: readonly { readonly name: string; readonly idFragment: string }[] = [
  { name: 'claude', idFragment: 'agents.claude' },
  { name: 'codex', idFragment: 'agents.codex' },
  { name: 'antigravity', idFragment: 'agents.antigravity' },
  { name: 'acp', idFragment: 'agents.acp' },
]

function matchAgentPlugin(text: string, allPlugins: readonly PluginMeta[]): string | undefined {
  for (const agent of AGENT_KEYWORDS) {
    if (text.includes(agent.name)) {
      const match = allPlugins.find((p) => p.id.includes(agent.idFragment))
      if (match) return match.id
    }
  }
  return undefined
}

function matchNamedPlugin(text: string, allPlugins: readonly PluginMeta[]): string | undefined {
  for (const p of allPlugins) {
    if (p.name && text.includes(p.name.toLowerCase())) {
      return p.id
    }
    const leaf = p.id.split('.').pop()?.toLowerCase()
    if (leaf && leaf.length > 2 && text.includes(leaf)) {
      return p.id
    }
  }
  return undefined
}

const CAPABILITY_MAP: readonly { readonly pattern: RegExp; readonly idFragment: string }[] = [
  { pattern: /proofread/, idFragment: 'proofreading' },
  { pattern: /transcrib|speech|audio/, idFragment: 'transcription' },
  { pattern: /ocr|recogni/, idFragment: 'ocr' },
]

function matchCapabilityPlugin(
  text: string,
  allPlugins: readonly PluginMeta[],
): string | undefined {
  for (const item of CAPABILITY_MAP) {
    if (item.pattern.test(text)) {
      const match = allPlugins.find((p) => p.id.includes(item.idFragment))
      if (match) return match.id
    }
  }
  return undefined
}

export function resolveTaskPlugin(
  task: Task,
  allPlugins: readonly PluginMeta[],
): string | undefined {
  const text = `${task.label} ${task.about} ${task.error}`.toLowerCase()
  return (
    matchAgentPlugin(text, allPlugins) ??
    matchNamedPlugin(text, allPlugins) ??
    matchCapabilityPlugin(text, allPlugins)
  )
}

export function resolveTaskAction(
  task: Task,
  allPlugins: readonly PluginMeta[],
): { label: string; pluginId: string } | undefined {
  if (!task.error || !isConfigurableError(task.error)) {
    return undefined
  }
  const pluginId = resolveTaskPlugin(task, allPlugins)
  if (!pluginId) {
    return undefined
  }
  return {
    label: 'Configure in settings',
    pluginId,
  }
}
