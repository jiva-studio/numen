/**
 * Resolves action links for failed background tasks.
 */
import type { Task } from '@/shared/notices/task'
import { WORDS as words } from '@/shared/words'

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

/** The settings action of a task whose plugin failed for want of configuration. */
export function resolveTaskAction(task: Task): { label: string; pluginId: string } | undefined {
  if (!task.pluginId || !isConfigurableError(task.error)) {
    return undefined
  }
  return { label: words.configure, pluginId: task.pluginId }
}
