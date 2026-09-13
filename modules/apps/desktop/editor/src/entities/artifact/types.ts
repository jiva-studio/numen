/**
 * What a model makes from one file of the vault, and what has become of it.
 *
 * Which model does the work follows from the file, so the window names the
 * artifact and never the producer.
 */

/**
 * What is made from one file of the vault, by what it is: text with the place
 * on the page each word stands at, text with the times it was said at, either
 * of those put right, the prose a page is written around, and the bytes of a
 * video kept to be played.
 *
 * What made it is another question. A transcript is a transcript whether a
 * model here wrote it down or a site published it with a video.
 */
export type Artifact =
  'ocr' | 'ocr.corrected' | 'transcript' | 'transcript.corrected' | 'article' | 'copy'

/**
 * What has become of one artifact: nothing has been made, a run over it waits
 * its turn behind another, a run is writing it now, a run stopped part way and
 * what it reached is on disk, the whole of it stands, a run found nothing to
 * write down, or a run could not read the file at all.
 *
 * The last two are what a run answered, and asking again gets the same until
 * the artifact is taken away.
 */
export type ArtifactState = 'none' | 'queued' | 'running' | 'stopped' | 'done' | 'empty' | 'failed'

/**
 * What a file carries, and what has become of each. Partial because which
 * artifacts a file carries follows from the file: a scan carries no
 * transcript, and a recording carries no text read.
 */
export type ArtifactStates = Partial<Record<Artifact, ArtifactState>>

/**
 * What asking for an artifact to be made answered: which artifact, what it now
 * is, and what a run said about a file it could not read. A build that cannot
 * make it at all answers nothing else, and the run is offered nowhere after
 * that.
 */
export type Outcome =
  | { readonly able: false }
  | {
      readonly able: true
      readonly of: Artifact
      readonly made: ArtifactState
      readonly error: string
    }

/** Which artifacts a file carries now. */
export interface ArtifactInspector {
  getArtifactStates(path: string): Promise<ArtifactStates>
}

/** Asks for one artifact of a file to be made. */
export interface ArtifactProducer {
  createArtifact(path: string, of: Artifact): Promise<Outcome>
}

/** Asks for a file's reading or transcript to be put right. */
export interface ArtifactCorrector {
  correctArtifact(path: string): Promise<Outcome>
}

/** Asks for what stands at the address a note points at. */
export interface ArtifactFetcher {
  fetchArtifact(path: string): Promise<Outcome>
}

/** Takes an artifact off the disk. */
export interface ArtifactDeleter {
  deleteTranscript(path: string): Promise<boolean>
  deleteCopy(path: string): Promise<boolean>
}

/** Everything a caller asks of the runs, for a caller that asks all of it. */
export interface ArtifactRunner
  extends
    ArtifactInspector,
    ArtifactProducer,
    ArtifactCorrector,
    ArtifactFetcher,
    ArtifactDeleter {}
