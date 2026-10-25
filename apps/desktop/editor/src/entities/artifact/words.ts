/**
 * What the window says about a run, in the words of the thing being run over: a
 * recording is transcribed, an address is fetched.
 */
// Stryker disable StringLiteral: interface copy
import type { ArtifactState, BuiltInArtifact } from './types'

export const FETCHED: Record<ArtifactState, string> = {
  none: 'The address published none of what was asked for.',
  queued: 'This address is in line, behind the one being fetched now.',
  running: 'This address is being fetched now.',
  stopped: 'Fetching this address stopped part way.',
  done: 'Fetched what is at this address.',
  empty: 'The address published none of what was asked for.',
  failed: 'This address could not be fetched:',
}

export const BUILT_IN_WORDS: Record<BuiltInArtifact, Record<ArtifactState, string>> = {
  transcript: {
    none: 'This recording has not been transcribed.',
    queued: 'This recording is in line, behind the one being transcribed now.',
    running: 'This recording is being transcribed now.',
    stopped: 'Transcribing this recording stopped part way.',
    done: 'This recording has already been transcribed.',
    empty: 'There is no speech in this recording.',
    failed: 'This recording could not be opened:',
  },
  'transcript.corrected': {
    none: 'Nothing has been transcribed here, so there is nothing to proofread.',
    queued: 'This transcript is in line, behind the one being put right now.',
    running: 'This transcript is being put right now.',
    stopped: 'Putting this transcript right stopped part way.',
    done: 'This transcript has already been put right.',
    empty: 'There were no words in this transcript to put right.',
    failed: 'This transcript could not be put right:',
  },
  copy: {
    none: 'No copy of this is on this disk.',
    queued: 'This is in line, behind the one being fetched now.',
    running: 'This is being fetched now.',
    stopped: 'Fetching this stopped part way.',
    done: 'A copy of this is on this disk.',
    empty: 'There is nothing at this address to copy.',
    failed: 'This was not copied:',
  },
  article: {
    none: 'Nothing has been fetched from this address.',
    queued: 'This address is in line, behind the one being fetched now.',
    running: 'This address is being fetched now.',
    stopped: 'Fetching this address stopped part way.',
    done: 'What is at this address has already been fetched.',
    empty: 'This address publishes none of what was asked for.',
    failed: 'This address could not be reached:',
  },
}
