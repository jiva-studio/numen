/** What a recording tab asks of the application, and what comes back. */
import type { ArtifactStates } from '@/shared/artifacts'
import type { Span } from '@/shared/span'
import type { Cue } from './lib/cues'

/** The text fetched or heard, and whether it may be written over. */
export interface Transcript {
  /** The words with the times they were said at, empty for prose nothing timed. */
  readonly cues: readonly Cue[]
  /** The prose a page is written around, empty for words with times. */
  readonly prose: string
  /** False while a run writing the text holds it. */
  readonly isEditable: boolean
}

/**
 * What a file is, for whatever plays it: how long it runs, in milliseconds,
 * and where its bytes come from.
 *
 * A recording nothing has listened to reaches nowhere, and the player it is
 * loaded into is what then says how long it runs.
 */
export interface RecordingSummary {
  readonly duration: number
  /** Where it is played from, as the application answers it. */
  readonly mediaUrl: string
  /**
   * What it is played as. The application says: what counts as a recording is
   * its to decide, and a url with no copy on this disk is a page to frame.
   */
  readonly mediaType: string
  /** The web address a url points at, and nothing on every other source. */
  readonly url: string
}

/** Everything a recording tab asks of the application. */
export interface Recordings {
  /** How long the recording runs, and how much of it has been written down. */
  getSummary(path: string): Promise<RecordingSummary>
  /** What the file carries, which says which of the two texts to read. */
  getTaskStates(path: string): Promise<ArtifactStates>
  /** The words heard in the recording, in the order they were spoken. */
  readTranscript(path: string): Promise<Transcript>
  /** The prose the page is written around, where nothing timed it. */
  readArticle(path: string): Promise<Transcript>
  /** The words as a person has edited them, kept against the recording. */
  writeTranscript(path: string, cues: readonly Cue[]): Promise<void>
  /**
   * The millisecond a span of the words written down is played from, and
   * nothing where no cue holds it.
   */
  findCueTime(path: string, span: Span): Promise<number | null>
}
