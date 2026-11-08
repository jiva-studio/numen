/** Every row the application and the shared commands place in a menu is drawn with an icon. */
import { describe, expect, it } from 'vitest'
import {
  createWhenReader,
  getTargetValues,
  SHARED_COMMANDS,
  type CommandTarget,
} from '@numen/plugin-host'
import { createCommands } from '@/features/command-palette/lib/commands/table'
import { WORDS } from '@/shared/words'
import { APP_SLOT_ENTRIES } from './appPlacements'
import { createSlotCommandFinder } from './pluginSlots'

const NOTE: CommandTarget = {
  path: 'tide.md',
  title: 'tide',
  isFolder: false,
  others: [],
  tab: '',
  noteType: 'note',
}

const findCommand = createSlotCommandFinder({
  whenReader: createWhenReader(),
  getPluginCommands: () => [],
  getDeclaredCommands: () => [],
  getEditorCommands: () => [
    ...createCommands(WORDS),
    ...['detach', 'unlink', 'link', 'moveToWindow'].map((name) => ({
      id: `app:tab.${name}`,
      text: name,
      when: 'true',
    })),
    { id: 'app:version.delete', text: 'Delete text', when: 'true' },
  ],
  hasPluginItems: (command) => command === 'app:version.delete',
  getHandlerDeclarations: () =>
    SHARED_COMMANDS.map((one) => ({
      pluginId: 'org.example',
      command: one.id,
      title: one.title,
      hasItems: false,
    })),
  getWhenValues: (target) => getTargetValues(target, {}),
})

describe('the icons of the rows of a menu', () => {
  it.each(APP_SLOT_ENTRIES.map((one) => one.command))(
    'the application command %s has one',
    (command) => {
      expect(findCommand(command, NOTE)?.icon).toBeDefined()
    },
  )

  it.each(SHARED_COMMANDS.map((one) => one.id))('the shared command %s has one', (command) => {
    expect(findCommand(command, NOTE)?.icon).toBeDefined()
  })
})
