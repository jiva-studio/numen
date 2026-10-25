/**
 * The commands offered over the file in front, on what has been made from it.
 */
import { createArtifactOffer, isArtifactMissing } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

const isUrlDeleteTextOffered = createArtifactOffer(
  'deleteText',
  'url',
  (createdItem) => createdItem.transcript !== 'none',
)

export const createArtifactCommands = (words: Words): readonly Command[] => [
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
]

export const createDeletionCommands = (words: Words): readonly Command[] => [
  {
    id: 'deleteText',
    text: words.deleteText,
    group: 'file',
    needs: 'asking',
    isOffered: isUrlDeleteTextOffered,
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
    isOffered: createArtifactOffer(
      'deleteCopy',
      'url',
      (createdItem) => createdItem.copy === 'done',
    ),
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
