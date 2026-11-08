/**
 * The conditions of a window, each read once: a reader keeps every condition it read by its text, so a
 * condition drawn or evaluated again is not parsed again. A window makes one reader and hands it to
 * every place that evaluates conditions.
 */
import type { Result } from '@numen/plugin-sdk'
import { parseWhen, type WhenCondition } from './when'

/** Reads conditions, each text once. */
export interface WhenReader {
  /** The condition of the text, or why it cannot be read; the same answer for the same text. */
  read(text: string): Result<WhenCondition, string>
  /** The condition of the text; undefined where it cannot be read. */
  getCondition(text: string | undefined): WhenCondition | undefined
}

export function createWhenReader(): WhenReader {
  const conditions = new Map<string, Result<WhenCondition, string>>()
  const read = (text: string): Result<WhenCondition, string> => {
    const cached = conditions.get(text)
    if (cached) return cached
    const parsed = parseWhen(text)
    conditions.set(text, parsed)
    return parsed
  }
  return {
    read,
    getCondition: (text) => {
      const parsed = read(text ?? '')
      return parsed.ok ? parsed.value : undefined
    },
  }
}
