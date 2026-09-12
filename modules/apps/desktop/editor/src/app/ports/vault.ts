/** What the window asks of the vault about itself, and the streams it speaks on. */
import type { PathRename } from '@/shared/paths'
import type { Task } from '@/shared/notices/task'
import type { OpenTabs } from '@/entities/tab'

export interface VaultPort {
  getInitialOpenPath(): Promise<{ path: string } | null>
  state(): Promise<{
    /** The identity the folder carries, which is how the vault is asked for again. */
    id: string
    /** What the person calls the vault. */
    name: string
    /** The folder the vault sits in, absolute on this machine. */
    path: string
    /** How far reading the vault has got. */
    scan: {
      isReady: boolean
      failureReason: string
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
   * Why an agent cannot be reached, and nothing where one can. Which agent
   * answers is the installation's, so it is asked of the agent and not of the
   * vault the window is showing.
   */
  agentUnreachable(): Promise<string>
  changes(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    shouldReload: boolean
    renamed: readonly PathRename[]
  }>
  /**
   * Everything the application is doing behind the window, for as long as the
   * window listens.
   *
   * The whole list arrives whenever any of it changes, and the first arrives at
   * once. It is a stream because work can begin without the window asking for
   * it: an agent is told to read a document, and this is where the person
   * watching sees it happen.
   */
  tasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
  /**
   * The places something else asked to be put in front of the person: a
   * source, and the span of its own text meant, counted in bytes. A length
   * of zero names the source and no place inside it.
   */
  focus(signal: AbortSignal): AsyncIterable<{
    path: string
    spans: readonly { from: number; to: number }[]
  }>
  /** What the person has open, said again whenever any of it changes. */
  writeOpenTabs(open: OpenTabs): Promise<void>
  /**
   * The window going, for as long as the client listens. The stream opens with
   * the token this client answers under.
   */
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Everything this client owed has been written. */
  flushed(token: string, owed?: 'written' | 'asking'): Promise<void>
}
