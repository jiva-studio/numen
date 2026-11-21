import { reactive } from 'vue'

/** The notes the plex draws as nodes, counted by the views mounted for them. */
export interface ShownPaths {
  readonly has: (path: string) => boolean
  readonly add: (path: string) => () => void
}

export function createShownPaths(): ShownPaths {
  const counts = reactive(new Map<string, number>())
  return {
    has: (path) => counts.has(path),
    add: (path) => {
      counts.set(path, (counts.get(path) ?? 0) + 1)
      return () => {
        const left = (counts.get(path) ?? 0) - 1
        if (left > 0) counts.set(path, left)
        else counts.delete(path)
      }
    },
  }
}
