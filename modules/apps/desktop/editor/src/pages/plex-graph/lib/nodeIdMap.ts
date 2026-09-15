/**
 * Map connecting note file paths to stable node IDs drawn on the plex graph.
 */
import { getRenamedPath, type PathRename } from '@/shared/paths'

export function createNodeIdMap() {
  let held = new Map<string, string>()
  let minted = 0

  const getNodeId = (path: string): string => {
    if (!path) return ''
    const id = held.get(path) ?? String(++minted)
    held.set(path, id)
    return id
  }

  const getNodePath = (id: string): string => {
    if (!id) return ''
    for (const [path, holds] of held) {
      if (holds === id) return path
    }
    return ''
  }

  const retainNodeIds = (ids: readonly string[]): void => {
    const drawing = new Set(ids)
    for (const [path, id] of held) {
      if (!drawing.has(id)) held.delete(path)
    }
  }

  const updateRenamedNodes = (renames: readonly PathRename[]): void => {
    const went = new Map<string, string>()
    for (const [path, id] of held) {
      went.set(getRenamedPath(renames, path) || path, id)
    }
    held = went
  }

  return {
    getNodeId,
    getNodePath,
    retainNodeIds,
    updateRenamedNodes,
  }
}

export type NodeIdMap = ReturnType<typeof createNodeIdMap>
