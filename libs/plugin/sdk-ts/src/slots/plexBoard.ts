// PlexBoardViewProps are the props the plex graph gives each view of `plexGraph.boardTop` and `plexGraph.boardBottom`: the key of the board, stable while its tab is open; the vault-relative path of the note in front of it, empty until it shows one; and `go`, which moves the board to a note as choosing the note in the board does.
export interface PlexBoardViewProps {
  readonly board: string
  readonly path: string
  readonly go: (path: string) => void
}
