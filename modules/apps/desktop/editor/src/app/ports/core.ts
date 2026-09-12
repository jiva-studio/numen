/**
 * Everything the window asks of the vault, and the words the vault speaks.
 *
 * Nothing here is about drawing: a kind translates these into what it holds,
 * and this is what every one of them starts from.
 */
import type { NotePort } from './notes'
import type { FilePort } from './files'
import type { PathRename } from '@/shared/paths'
import type { Task } from '@/shared/notices/task'
import type { Configuration, SettingEdit } from '@/entities/settings'
import type { HangingSettings } from '@/entities/settings'
import type { ReviewSettings } from '@/entities/settings'
import type { Attention } from '@/entities/tab'

export interface Core extends NotePort, FilePort {
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
  /**
   * What the person has open, said again whenever any of it changes. It is the
   * other direction to `focus`: a place is put in front of the person there,
   * and here the window says what is in front of them now.
   */
  setFocus(open: Attention): Promise<void>
  /**
   * Whether renaming either a note's title or the name of its file brings the
   * other into line, as the settings hold it.
   */
  getSyncEnabled(): Promise<boolean>
  /**
   * That setting written into the settings file. What could not be written, and
   * nothing where it was: the rename after this reads what was written.
   */
  setSyncEnabled(kept: boolean): Promise<string | null>
  /**
   * Whether a node in the plex hangs the parts of its note under the box, and
   * how many of them stand there at once, as the settings hold them.
   */
  getHangingSettings(): Promise<HangingSettings>
  /**
   * Those settings written into the settings file. What could not be written,
   * and nothing where it was. A count left out stands as it is.
   */
  setHangingSettings(hangs: boolean, parts?: number): Promise<string | null>
  /**
   * The hour a day of review begins at, on the clock on the wall, written as
   * `04:00`, and how late in the day the vault takes one. An hour past that is
   * refused.
   */
  getReviewSettings(): Promise<ReviewSettings>
  /**
   * That hour written into the settings file. What could not be written, and
   * nothing where it was.
   */
  setReviewSettings(starts: string): Promise<string | null>
  /** Every setting as it stands, and the models the settings offer. */
  getSettings(): Promise<Configuration>
  /**
   * Settings written into the settings file, together or not at all. A value
   * the settings could not be read out of again is refused, and what the file
   * holds is unchanged.
   */
  updateSettings(written: readonly SettingEdit[]): Promise<void>
  /** The settings file as its person wrote it, and where it stands. */
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  /**
   * The settings file replaced whole, with the bytes as they were typed. A file
   * the settings could not be read out of is refused, and what the file holds
   * is unchanged.
   *
   * Seen is the file as it was last read, and a file standing at anything else
   * is answered `changed` with nothing written. Nothing seen writes over
   * whatever the file holds.
   */
  saveSettingsFile(
    written: string,
    seen: string | null,
  ): Promise<{ readonly changed: boolean }>
  /**
   * The window going, for as long as the client listens. The stream opens with
   * the token this client answers under.
   */
  quitting(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Everything this client owed has been written. */
  flushed(token: string, owed?: 'written' | 'asking'): Promise<void>
}
