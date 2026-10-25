/** What a recording tab asks of the application, and what comes back. */
import type { Cue } from './lib/cues'

/** Where a recording is played from, and what it is played as. */
export interface RecordingSource {
  readonly mediaUrl: string
  readonly mediaType: string
}

/** A run of the transcript's text, as the window counts it. */
export interface Span {
  readonly from: number
  readonly to: number
}

/** What an entry of a file's menu does. */
export type MenuAction = 'recognise' | 'proofread' | 'deleteVersion'

/**
 * One entry of the menu the core makes for a file. `versionId` is the version it works on;
 * `isLive` says that version is the one shown and indexed now.
 */
export interface MenuEntry {
  readonly action: MenuAction
  readonly pluginId: string
  readonly versionId: string
  readonly isLive: boolean
}

/** Everything a recording tab asks of the application. */
export interface Recordings {
  /** Where the recording is played from. */
  getSource(path: string): Promise<RecordingSource>
  /** The words heard in the recording, in the order they were spoken. */
  readTranscript(path: string): Promise<readonly Cue[]>
  /** The words as a person has edited them, kept against the recording. */
  writeTranscript(path: string, cues: readonly Cue[]): Promise<void>
  /** The millisecond a span of the words is played from, and null where no cue holds it. */
  findCueTime(path: string, span: Span): Promise<number | null>
  /** What the file's menu offers about its text. */
  getMenu(path: string): Promise<readonly MenuEntry[]>
  /** Carries out one entry of the file's menu. */
  runMenuEntry(path: string, entry: MenuEntry): Promise<void>
  /**
   * Tells the handler whether work on the file is going, at once and on every change, until the
   * returned function is called.
   */
  watchWork(path: string, onChange: (isWorking: boolean) => void): () => void
}
