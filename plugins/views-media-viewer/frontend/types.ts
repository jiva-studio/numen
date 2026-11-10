/** The requests a recording tab makes to the application, and their results. */

/** One cue: its text and the milliseconds it covers. */
export interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

/** Where a recording is played from and what it is played as, or the page that frames its player. */
export interface RecordingSource {
  readonly mediaUrl: string
  readonly mediaType: string
  readonly pageUrl: string
}

/** A run of the transcript's text, as the window counts it. */
export interface TextRange {
  readonly start: number
  readonly end: number
}

/** All requests a recording tab makes to the application. */
export interface Recordings {
  /** Where the recording is played from. */
  getSource(path: string): Promise<RecordingSource>
  /** The words heard in the recording, in the order they were spoken. */
  readTranscript(path: string): Promise<readonly Cue[]>
  /** The prose of a live text without times, such as an article, and empty otherwise. */
  readArticle(path: string): Promise<string>
  /** The transcript as the user has edited it, saved against the recording. */
  writeTranscript(path: string, cues: readonly Cue[]): Promise<void>
  /** The millisecond a span of the words is played from, and null where no cue holds it. */
  findCueTime(path: string, span: TextRange): Promise<number | null>
  /** Whether a plugin creates the text of the file, through the shared command `app:file.createText`. */
  canCreateText(path: string): Promise<boolean>
  /** Runs the shared command that creates the text of the file. */
  createText(path: string): Promise<void>
  /**
   * Notifies the handler whether processing of the file is in progress, once
   * immediately and again on every change, until the returned function is called.
   */
  watchWork(path: string, onChange: (isWorking: boolean) => void): () => void
  /**
   * Notifies the handler when the vault reports a change to the file, such as a text version
   * deleted from outside the tab, until the returned function is called.
   */
  watchChanges(path: string, onChange: () => void): () => void
}
