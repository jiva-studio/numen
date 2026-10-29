/** The vault operations on files and folders available to the window. */
import type { ErrorCode } from '@/shared/errors'
import type { Entry, FileKind, Movement } from '@/entities/file'

export interface FilePort {
  /**
   * Kind is read from the vault itself, so a path not yet scanned returns what
   * exists there; a path with nothing at it is absent.
   */
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  /**
   * What one folder of the vault holds, in the order to draw it: folders first
   * and then files, each group by name with case ignored. The root is the empty
   * path, and hidden files are in none of the answers.
   */
  list(folder: string): Promise<readonly Entry[]>
  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what
   * it is called from now on, so a name changed within one folder is a move.
   */
  move(from: string, to: string): Promise<Movement>
  /** An empty folder. The folders above it are made with it. */
  createFolder(path: string): Promise<ErrorCode | null>
}
