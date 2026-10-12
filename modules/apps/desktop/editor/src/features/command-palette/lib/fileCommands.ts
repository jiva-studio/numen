/**
 * The commands offered over the file in front, on what has been made from it.
 */
import { createArtifactOffer, isArtifactMissing } from './offered'
import type { Command, CommandTarget, RunSupport } from '../types'
import type { Words } from '../words'

const isRecordingProofreadOffered = createArtifactOffer(
  'proofread',
  'recording',
  (made) => made.transcript === 'done' && isArtifactMissing(made['transcript.corrected']),
)

const isBookProofreadOffered = createArtifactOffer(
  'proofread',
  'book',
  (made) => made.ocr === 'done' && isArtifactMissing(made['ocr.corrected']),
)

const isProofreadOffered = (at: CommandTarget, runs: RunSupport): boolean =>
  isRecordingProofreadOffered(at, runs) || isBookProofreadOffered(at, runs)

const isRecordingDeleteTextOffered = createArtifactOffer(
  'deleteText',
  'recording',
  (made) => made.transcript !== 'none',
)

const isUrlDeleteTextOffered = createArtifactOffer(
  'deleteText',
  'url',
  (made) => made.transcript !== 'none',
)

const isDeleteTextOffered = (at: CommandTarget, runs: RunSupport): boolean =>
  isRecordingDeleteTextOffered(at, runs) || isUrlDeleteTextOffered(at, runs)

export const fileCommandsOf = (words: Words): readonly Command[] => [
  {
    id: 'transcribe',
    text: words.transcribe,
    group: 'file',
    isOffered: createArtifactOffer('transcribe', 'recording', (made) =>
      isArtifactMissing(made.transcript),
    ),
  },
  {
    id: 'recognise',
    text: words.recognise,
    group: 'file',
    isOffered: createArtifactOffer('recognise', 'book', (made) => isArtifactMissing(made.ocr)),
  },
  {
    id: 'downloadText',
    text: words.downloadText,
    group: 'file',
    isOffered: createArtifactOffer(
      'downloadText',
      'url',
      (made) => made.transcript !== undefined || made.article !== undefined,
    ),
  },
  {
    id: 'downloadCopy',
    text: words.downloadCopy,
    group: 'file',
    // Only a note pointing at a video carries a copy at all, so the row being
    // there is what says this file is one. An hour of video on somebody's disk
    // is asked for by hand, and one already here is not asked for again.
    isOffered: (at, runs) =>
      at.isReady &&
      runs.canRun('downloadCopy') &&
      at.made.copy !== undefined &&
      isArtifactMissing(at.made.copy),
  },
  {
    id: 'proofread',
    text: words.proofread,
    group: 'file',
    // There is nothing to put right until a model has produced text, and
    // nothing to put right again once it has been put right.
    isOffered: isProofreadOffered,
  },
  {
    id: 'deleteText',
    text: words.deleteText,
    group: 'file',
    needs: 'asking',
    // Everything one run of listening left goes, so a run that stopped part way
    // and a recording that gave no words are both taken away here. A url
    // carries a transcript the same way, and deletes it here.
    isOffered: isDeleteTextOffered,
    answers: {
      keeps: words.keepsTranscript,
      kept: words.kept,
      action: words.deletes,
      then: words.deleted,
    },
  },
  {
    id: 'deleteCopy',
    text: words.deleteCopy,
    group: 'file',
    needs: 'asking',
    // Only a url carries a copy, and only one that stands has anything to take.
    isOffered: createArtifactOffer('deleteCopy', 'url', (made) => made.copy === 'done'),
    answers: {
      keeps: words.keepsCopy,
      kept: words.kept,
      action: words.deletes,
      then: words.deletedCopy,
    },
  },
]
