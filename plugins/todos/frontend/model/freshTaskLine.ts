import type { Task } from '../types'

/** The line of the task the note holds now: the one with the same raw text nearest to the stored line. */
export function freshTaskLine(fresh: readonly Task[], target: Pick<Task, 'line' | 'raw'>): number {
  let best = target.line
  let distance = Number.POSITIVE_INFINITY
  for (const one of fresh) {
    if (one.raw !== target.raw) continue
    const away = Math.abs(one.line - target.line)
    if (away < distance) {
      best = one.line
      distance = away
    }
  }
  return best
}
