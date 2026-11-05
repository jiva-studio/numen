/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { COMMAND } from '@/shared/commandIds'
import { createRowKeys } from '../chords/chords'
import { isAlwaysAvailable, isFileChosen } from '../commands/availability'
import type { Command } from '../../types'
import type { Words } from '../../words'

export const createWindowCreationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.createNote,
    text: words.newNote,
    ...createRowKeys(COMMAND.createNote, agent),
    group: 'window',
    iconName: 'file-plus',
    input: 'name',
    isAvailable: (target) => target.isReady,
  },
]

export const createWindowNavigationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.closeTab,
    text: words.close,
    ...createRowKeys(COMMAND.closeTab, agent),
    group: 'window',
    isAvailable: (target) => target.tab !== '',
  },
  {
    id: COMMAND.openFile,
    text: words.readDocument,
    group: 'file',
    isAvailable: isFileChosen,
  },
  {
    id: COMMAND.searchVault,
    text: words.searchVault,
    keys: words.findKeys,
    group: 'window',
    isAvailable: isAlwaysAvailable,
  },
]

export const createWindowDisplayCommands = (words: Words): readonly Command[] => [
  {
    id: COMMAND.chooseTheme,
    text: words.appearance,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.chooseMode,
    text: words.chooseMode,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.chooseInterfaceSize,
    text: words.chooseInterfaceSize,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.chooseTextSize,
    text: words.chooseTextSize,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
]

export const createWindowFeatureCommands = (words: Words): readonly Command[] => [
  {
    id: COMMAND.chooseTitleSync,
    text: words.chooseTitleSync,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.choosePerformanceProfile,
    text: words.performanceProfile,
    group: 'settings',
    input: 'choice',
    isAvailable: isAlwaysAvailable,
  },
  {
    id: COMMAND.openSettings,
    text: words.openSettings,
    group: 'settings',
    isAvailable: isAlwaysAvailable,
  },
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
