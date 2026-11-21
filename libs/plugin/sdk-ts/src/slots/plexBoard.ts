// PlexBoardViewProps are the props the plex graph gives each view of `plexGraph.boardTop` and `plexGraph.boardBottom`.
export interface PlexBoardViewProps {
  // The key of the board, stable for as long as the board is open.
  readonly board: string
  // The vault-relative path of the note in front of the board; empty until the board shows one.
  readonly path: string
  // Moves the board to a note, as choosing the note in the board does.
  readonly go: (path: string) => void
}
