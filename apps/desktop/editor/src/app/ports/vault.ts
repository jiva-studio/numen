import type { PathRename } from '@/shared/paths'
import type { Task } from '@/shared/notices/task'
import type { OpenTabs } from '@/entities/tab'

export interface VaultPort {
  getInitialOpenPath(): Promise<{ path: string } | null>
  state(): Promise<{
    /** The folder's identity, used to request the vault again. */
    id: string
    /** The user-visible vault name. */
    name: string
    path: string
    /** How far indexing the vault has got. */
    scan: {
      isReady: boolean
      error: string
      unwatchedPath: string
    }
    /** How far searching it by meaning has got. */
    coverage: {
      chunkCount: bigint
      embeddedCount: bigint
      isEmbedding: boolean
    }
  }>
  /**
   * Why an agent cannot be reached, or an empty string when it can. Which agent answers is
   * determined by the installation, so the check is made on the agent.
   */
  agentUnreachable(): Promise<string>
  watchVaultChanges(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    shouldReload: boolean
    renamed: readonly PathRename[]
  }>
  /**
   * Everything the application is doing in the background, for as long as the caller listens.
   *
   * The whole list is sent whenever any of it changes, and the first list is sent immediately.
   * It is a stream because work can begin without a request from the window: for example an
   * agent reading a document, which the user sees here.
   */
  watchTasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
  /**
   * The places another component asked to show to the user: a source and the ranges of its text,
   * counted in bytes. A length of zero names the source and no place inside it.
   */
  watchFocus(signal: AbortSignal): AsyncIterable<{
    path: string
    ranges: readonly { start: number; end: number }[]
  }>
  /** Reports the currently open tabs; called again whenever they change. */
  writeOpenTabs(open: OpenTabs): Promise<void>
  /** The tabs last written for the vault in the window, kept by the core across restarts. */
  readOpenTabs(): Promise<OpenTabs>
  /**
   * The window quitting, for as long as the client listens. The stream opens with the token this
   * client replies under.
   */
  watchQuit(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Reports that this client has written all pending data. */
  reportFlush(token: string, owed?: 'written' | 'asking'): Promise<void>
  /** Detach a tab into a new window at the given screen coordinates. */
  detachTab(tabId: string, position: { x: number; y: number }): Promise<void>
}
