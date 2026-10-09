/**
 * Map connecting note file paths to stable node IDs drawn on the plex graph.
 */
import { getRenamedPath, type PathRename } from '@/shared/paths'

export function createNodeIdMap() {
  let nodeIdsByPath = new Map<string, string>()
  let nextId = 0

  const getNodeId = (path: string): string => {
    if (!path) return ''
    const id = nodeIdsByPath.get(path) ?? String(++nextId)
    nodeIdsByPath.set(path, id)
    return id
  }

  const getNodePath = (id: string): string => {
    if (!id) return ''
    for (const [path, mappedId] of nodeIdsByPath) {
      if (mappedId === id) return path
    }
    return ''
  }

  const retainNodeIds = (ids: readonly string[]): void => {
    const drawing = new Set(ids)
    for (const [path, id] of nodeIdsByPath) {
      if (!drawing.has(id)) nodeIdsByPath.delete(path)
    }
  }

  const updateRenamedNodes = (renames: readonly PathRename[]): void => {
    const updatedMap = new Map<string, string>()
    for (const [path, id] of nodeIdsByPath) {
      updatedMap.set(getRenamedPath(renames, path) || path, id)
    }
    nodeIdsByPath = updatedMap
  }

  return {
    getNodeId,
    getNodePath,
    retainNodeIds,
    updateRenamedNodes,
  }
}

export type NodeIdMap = ReturnType<typeof createNodeIdMap>
