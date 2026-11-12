/**
 * The commands the palette itself owns: close the tab in front, search the vault, open the settings.
 */
import { type Command } from '@/entities/command'
import { COMMAND, createRowKeys, withWhen } from '@/shared/model/command'
import type { Words } from '../../words'

export const createWindowCommands = (words: Words, agent: string): readonly Command[] => [
  {
    id: COMMAND.closeTab,
    text: words.close,
    ...createRowKeys(COMMAND.closeTab, agent),
    group: 'window',
    ...withWhen('window.isTabOpen'),
  },
  {
    id: COMMAND.searchVault,
    text: words.searchVault,
    keys: words.findKeys,
    group: 'window',
    ...withWhen(''),
  },
  {
    id: COMMAND.openSettings,
    text: words.openSettings,
    group: 'settings',
    iconName: 'settings',
    ...withWhen(''),
  },
]
