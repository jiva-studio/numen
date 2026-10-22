/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { createRowKeys } from './chords'
import { isAlwaysOffered } from './availability'
import type { Command } from '../types'
import type { Words } from '../words'

export const createWindowCreationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'note',
    text: words.newNote,
    ...createRowKeys('note', agent),
    group: 'window',
    needs: 'naming',
    isOffered: (target) => target.isReady,
  },
  {
    id: 'deck',
    text: words.newDeck,
    group: 'window',
    needs: 'naming',
    isOffered: (target) => target.isReady,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    group: 'window',
    needs: 'naming',
    isOffered: (target) => target.isReady,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    group: 'window',
    needs: 'naming',
    isOffered: (target) => target.isReady,
  },
  {
    id: 'importUrl',
    text: words.importUrl,
    group: 'window',
    needs: 'address',
    isOffered: (target) => target.isReady,
  },
]

export const createWindowNavigationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'plex',
    text: words.newPlex,
    ...createRowKeys('plex', agent),
    group: 'window',
    isOffered: isAlwaysOffered,
  },
  { id: 'files', text: words.files, group: 'window', isOffered: isAlwaysOffered },
  {
    id: 'agent',
    text: words.newAgent,
    ...createRowKeys('agent', agent),
    group: 'window',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'close',
    text: words.close,
    ...createRowKeys('close', agent),
    group: 'window',
    isOffered: (target) => target.tab !== '',
  },
  {
    id: 'find',
    text: words.find,
    keys: words.findKeys,
    group: 'window',
    isOffered: isAlwaysOffered,
  },
]

export const createWindowDisplayCommands = (words: Words): readonly Command[] => [
  {
    id: 'appearance',
    text: words.appearance,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'mode',
    text: words.mode,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'textScale',
    text: words.textScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
]

export const createWindowFeatureCommands = (words: Words): readonly Command[] => [
  {
    id: 'syncing',
    text: words.syncing,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'nesting',
    text: words.nesting,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'headings',
    text: words.headings,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'performanceProfile',
    text: words.performanceProfile,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'indexingModel',
    text: words.indexingModel,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  { id: 'settings', text: words.settings, group: 'window', isOffered: isAlwaysOffered },
]

export const createWindowSettingsCommands = (words: Words): readonly Command[] => [
  ...createWindowDisplayCommands(words),
  ...createWindowFeatureCommands(words),
]

export const createWindowCommands = (words: Words, agent: string): readonly Command[] => [
  ...createWindowCreationCommands(words, agent),
  ...createWindowNavigationCommands(words, agent),
  ...createWindowSettingsCommands(words),
]
