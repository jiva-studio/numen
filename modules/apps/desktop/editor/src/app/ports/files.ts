/** What the window asks of the vault about files and folders. */
import type { ErrorCode } from '@/shared/errors'
import type { CreateResult, Entry, FileKind, Movement } from '@/entities/file'

export interface FilePort {
  /**
   * What stands at each of those paths, by the path it was asked about. The
   * kind comes off the vault itself, so a path nothing has scanned is answered
   * with what stands there; a path with nothing at it is absent.
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
  /**
   * The file a web address is kept in, named by the address until a fetch says
   * what is there.
   */
  createUrl(url: string, folder: string): Promise<CreateResult>
}
