import type { Stats } from '../types'

/** Whether chunks still wait for a vector, which is when the index is being filled. */
export function isPending(stats: Pick<Stats, 'waiting'>): boolean {
  return stats.waiting > 0
}
