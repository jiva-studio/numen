import type { Task } from '../types'

/** The task the note holds now: the one with the same raw text nearest to the stored line. */
export function createTaskLine(
  fresh: readonly Task[],
  target: Pick<Task, 'line' | 'raw'>,
): Task | undefined {
  let best: Task | undefined
  let distance = Number.POSITIVE_INFINITY
  for (const one of fresh) {
    if (one.raw !== target.raw) continue
    const away = Math.abs(one.line - target.line)
    if (away < distance) {
      best = one
      distance = away
    }
  }
  return best
}
