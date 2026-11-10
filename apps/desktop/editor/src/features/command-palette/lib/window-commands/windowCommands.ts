/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { COMMAND, ROW } from '@/shared/commandIds'
import { createRowKeys } from '../keys/keys'
import { withWhen } from '../commands/availability'
import type { Command } from '@/entities/command'
import type { Words } from '../../words'

export const createWindowCreationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.createNote,
    text: words.newNote,
    ...createRowKeys(COMMAND.createNote, agent),
    group: 'window',
    iconName: 'file-plus',
    input: 'name',
    canOmitNameOverFolder: true,
    ...withWhen('window.isVaultReady'),
  },
  {
    id: COMMAND.createFolder,
    text: words.newFolder,
    group: 'window',
    input: 'name',
    canOmitNameOverFolder: true,
    ...withWhen('window.isVaultReady'),
  },
]

export const createWindowNavigationCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.closeTab,
    text: words.close,
    ...createRowKeys(COMMAND.closeTab, agent),
    group: 'window',
    ...withWhen('window.isTabOpen'),
  },
  {
    id: ROW.openDocument,
    command: COMMAND.openFile,
    text: words.readDocument,
    group: 'file',
    ...withWhen('target.isFile && !window.isTabOpen'),
  },
  {
    id: COMMAND.searchVault,
    text: words.searchVault,
    keys: words.findKeys,
    group: 'window',
    ...withWhen(''),
  },
]

/** The settings the editor offers are made from their declarations; only the tab is a command here. */
export const createWindowSettingsCommands = (words: Words): readonly Command[] => [
  {
    id: COMMAND.openSettings,
    text: words.openSettings,
    group: 'settings',
    iconName: 'settings',
    ...withWhen(''),
  },
]

export const createWindowCommands = (words: Words, agent: string): readonly Command[] => [
  ...createWindowCreationCommands(words, agent),
  ...createWindowNavigationCommands(words, agent),
  ...createWindowSettingsCommands(words),
]
