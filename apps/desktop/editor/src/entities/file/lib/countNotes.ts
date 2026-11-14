import type { Entry } from '../types'

/** How many notes the vault holds, counted by listing the root and every folder below it. */
export async function countNotes(
  list: (folder: string) => Promise<readonly Entry[]>,
): Promise<number> {
  let count = 0
  const pending = ['']
  for (let folder = pending.pop(); folder !== undefined; folder = pending.pop()) {
    for (const one of await list(folder)) {
      if (one.isFolder) pending.push(one.path)
      else if (one.isNote) count += 1
    }
  }
  return count
}
