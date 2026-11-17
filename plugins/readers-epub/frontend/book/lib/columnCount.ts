import { columnPitch } from './columnPitch'
import type { Flow } from './spread'

/**
 * How many columns the text evaluates to. The count is taken off the whole run and
 * rounded once: a trailing gap adds no column, and nothing drifts over a long
 * document.
 */
export function columnsInAll(flow: Flow): number {
  const pitch = columnPitch(flow)
  if (pitch <= 0 || flow.contentWidth <= 0) return 0
  return Math.max(1, Math.round(flow.contentWidth / pitch))
}
