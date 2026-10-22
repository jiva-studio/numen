/**
 * The commands offered over the file in front, on what has been made from it.
 */
import { createArtifactOffer, isArtifactMissing } from './availability'
import type { Command, CommandTarget, RunSupport } from '../types'
import type { Words } from '../words'

const isRecordingProofreadOffered = createArtifactOffer(
  'proofread',
  'recording',
  (createdItem) => createdItem.transcript === 'done' && isArtifactMissing(createdItem['transcript.corrected']),
)

const isBookProofreadOffered = createArtifactOffer(
  'proofread',
  'book',
  (createdItem) => createdItem.ocr === 'done' && isArtifactMissing(createdItem['ocr.corrected']),
)

const isProofreadOffered = (target: CommandTarget, runs: RunSupport): boolean =>
  isRecordingProofreadOffered(target, runs) || isBookProofreadOffered(target, runs)

const isRecordingDeleteTextOffered = createArtifactOffer(
  'deleteText',
  'recording',
  (createdItem) => createdItem.transcript !== 'none',
)

const isUrlDeleteTextOffered = createArtifactOffer(
  'deleteText',
  'url',
  (createdItem) => createdItem.transcript !== 'none',
)

const isDeleteTextOffered = (target: CommandTarget, runs: RunSupport): boolean =>
  isRecordingDeleteTextOffered(target, runs) || isUrlDeleteTextOffered(target, runs)

export const createArtifactCommands = (words: Words): readonly Command[] => [
  {
    id: 'transcribe',
    text: words.transcribe,
    group: 'file',
    isOffered: createArtifactOffer('transcribe', 'recording', (createdItem) =>
      isArtifactMissing(createdItem.transcript),
    ),
  },
  {
    id: 'recognise',
    text: words.recognise,
    group: 'file',
    isOffered: createArtifactOffer('recognise', 'book', (createdItem) => isArtifactMissing(createdItem.ocr)),
  },
  {
    id: 'downloadText',
    text: words.downloadText,
    group: 'file',
    isOffered: createArtifactOffer(
      'downloadText',
      'url',
      (createdItem) => createdItem.transcript !== undefined || createdItem.article !== undefined,
    ),
  },
  {
    id: 'downloadCopy',
    text: words.downloadCopy,
    group: 'file',
    isOffered: (target, runs) =>
      target.isReady &&
      runs.canRun('downloadCopy') &&
      target.made.copy !== undefined &&
      isArtifactMissing(target.made.copy),
  },
  {
    id: 'proofread',
    text: words.proofread,
    group: 'file',
    isOffered: isProofreadOffered,
  },
]

export const createDeletionCommands = (words: Words): readonly Command[] => [
  {
    id: 'deleteText',
    text: words.deleteText,
    group: 'file',
    needs: 'asking',
    isOffered: isDeleteTextOffered,
    answers: {
      keeps: words.keepsTranscript,
      retainedNotice: words.retainedNotice,
      action: words.deletes,
      detail: words.deleted,
    },
  },
  {
    id: 'deleteCopy',
    text: words.deleteCopy,
    group: 'file',
    needs: 'asking',
    isOffered: createArtifactOffer('deleteCopy', 'url', (createdItem) => createdItem.copy === 'done'),
    answers: {
      keeps: words.keepsCopy,
      retainedNotice: words.retainedNotice,
      action: words.deletes,
      detail: words.deletedCopy,
    },
  },
]

export const createFileCommands = (words: Words): readonly Command[] => [
  ...createArtifactCommands(words),
  ...createDeletionCommands(words),
]
