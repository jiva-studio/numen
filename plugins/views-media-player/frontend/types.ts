/** The requests a recording tab makes to the application, and their results. */
import type { Cue } from './lib/cues'

/** Where a recording is played from, and what it is played as. */
export interface RecordingSource {
  readonly mediaUrl: string
  readonly mediaType: string
}

/** A run of the transcript's text, as the window counts it. */
export interface TextRange {
  readonly start: number
  readonly end: number
}

/** What an entry of a file's menu does: a plugin makes text, or the owner of a version deletes it. */
export type MenuAction = 'createText' | 'deleteVersion'

/**
 * One entry of the menu the core builds for a file. `versionId` is the version
 * it applies to, and empty for text created from the file; `isLive` marks that
 * version as the one currently displayed and indexed.
 */
export interface MenuEntry {
  readonly action: MenuAction
  readonly pluginId: string
  readonly versionId: string
  readonly isLive: boolean
}

/** All requests a recording tab makes to the application. */
export interface Recordings {
  /** Where the recording is played from. */
  getSource(path: string): Promise<RecordingSource>
  /** The words heard in the recording, in the order they were spoken. */
  readTranscript(path: string): Promise<readonly Cue[]>
  /** The transcript as the user has edited it, saved against the recording. */
  writeTranscript(path: string, cues: readonly Cue[]): Promise<void>
  /** The millisecond a span of the words is played from, and null where no cue holds it. */
  findCueTime(path: string, span: TextRange): Promise<number | null>
  /** What the file's menu offers about its text. */
  getMenu(path: string): Promise<readonly MenuEntry[]>
  /** Carries out one entry of the file's menu. */
  runMenuEntry(path: string, entry: MenuEntry): Promise<void>
  /**
   * Notifies the handler whether processing of the file is in progress, once
   * immediately and again on every change, until the returned function is called.
   */
  watchWork(path: string, onChange: (isWorking: boolean) => void): () => void
}
