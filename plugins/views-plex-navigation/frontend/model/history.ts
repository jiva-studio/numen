import type { BoardKey } from '../types'

interface Trail {
  readonly nodes: string[]
  cursor: number
}

export interface BoardHistories {
  visit(board: BoardKey, node: string): void
  back(board: BoardKey): string | null
  forward(board: BoardKey): string | null
  canBack(board: BoardKey): boolean
  canForward(board: BoardKey): boolean
  forget(board: BoardKey): void
}

export function createBoardHistories(): BoardHistories {
  const trails = new Map<BoardKey, Trail>()

  const getTrail = (board: BoardKey): Trail => {
    const found = trails.get(board)
    if (found) return found
    const made: Trail = { nodes: [], cursor: -1 }
    trails.set(board, made)
    return made
  }

  const canBack = (board: BoardKey) => getTrail(board).cursor > 0
  const canForward = (board: BoardKey) => {
    const trail = getTrail(board)
    return trail.cursor < trail.nodes.length - 1
  }

  return {
    visit(board, node) {
      const trail = getTrail(board)
      if (trail.nodes[trail.cursor] === node) return
      trail.nodes.length = trail.cursor + 1
      trail.nodes.push(node)
      trail.cursor += 1
    },
    back(board) {
      if (!canBack(board)) return null
      const trail = getTrail(board)
      trail.cursor -= 1
      return trail.nodes[trail.cursor] ?? null
    },
    forward(board) {
      if (!canForward(board)) return null
      const trail = getTrail(board)
      trail.cursor += 1
      return trail.nodes[trail.cursor] ?? null
    },
    canBack,
    canForward,
    forget: (board) => {
      trails.delete(board)
    },
  }
}
