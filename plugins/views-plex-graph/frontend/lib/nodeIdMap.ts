/**
 * Map connecting note file paths to stable node IDs drawn on the plex graph.
 */
import { getRenamedPath, type PathRename } from '@numen/plugin-sdk'

export function createNodeIdMap() {
  let nodeIdsByPath = new Map<string, string>()
  let pathsByNodeId = new Map<string, string>()
  let nextId = 0

  const getNodeId = (path: string): string => {
    if (!path) return ''
    let id = nodeIdsByPath.get(path)
    if (!id) {
      id = String(++nextId)
      nodeIdsByPath.set(path, id)
      pathsByNodeId.set(id, path)
    }
    return id
  }

  const getNodePath = (id: string): string => pathsByNodeId.get(id) ?? ''

  const updateRenamedNodes = (renames: readonly PathRename[]): void => {
    const updatedByPath = new Map<string, string>()
    const updatedById = new Map<string, string>()
    for (const [path, id] of nodeIdsByPath) {
      const nextPath = getRenamedPath(renames, path) || path
      updatedByPath.set(nextPath, id)
      updatedById.set(id, nextPath)
    }
    nodeIdsByPath = updatedByPath
    pathsByNodeId = updatedById
  }

  return {
    getNodeId,
    getNodePath,
    updateRenamedNodes,
  }
}

export type NodeIdMap = ReturnType<typeof createNodeIdMap>
