/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { createRowKeys } from '../chords/chords'
import { isAlwaysAvailable } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createWindowCreationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'note',
    text: words.newNote,
    ...createRowKeys('note', agent),
    group: 'window',
    needs: 'naming',
    isAvailable: (target) => target.isReady,
  },
]

export const createWindowNavigationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'close',
    text: words.close,
    ...createRowKeys('close', agent),
    group: 'window',
    isAvailable: (target) => target.tab !== '',
  },
  {
    id: 'find',
    text: words.find,
    keys: words.findKeys,
    group: 'window',
    isAvailable: isAlwaysAvailable,
  },
]

export const createWindowDisplayCommands = (words: Words): readonly Command[] => [
  {
    id: 'appearance',
    text: words.appearance,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'mode',
    text: words.mode,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'textScale',
    text: words.textScale,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
]

export const createWindowFeatureCommands = (words: Words): readonly Command[] => [
  {
    id: 'syncing',
    text: words.syncing,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: 'performanceProfile',
    text: words.performanceProfile,
    group: 'settings',
    needs: 'choosing',
    isAvailable: isAlwaysAvailable,
  },
  { id: 'settings', text: words.settings, group: 'settings', isAvailable: isAlwaysAvailable },
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
