/** A shared command that several plugins place in one menu, as the file tree reads the menu. */
import { describe, expect, it } from 'vitest'
import {
  createWhenReader,
  createSlotRegistry,
  getTargetValues,
  type CommandTarget,
  type SlotItem,
} from '@numen/plugin-loader'
import { createSlotCommandFinder, readSlotEntries } from './pluginSlots'

const PAGES = 'org.example.pages'
const SCANS = 'org.example.scans'

const PDF: CommandTarget = {
  path: 'books/tide.pdf',
  title: 'tide',
  isFolder: false,
  others: [],
  tab: '',
}

const describeItem = (item: SlotItem): string =>
  item.kind === 'command' ? item.command : item.kind

describe('a shared command placed by several plugins', () => {
  it('shows once in the menu', () => {
    const findCommand = createSlotCommandFinder({
      whenReader: createWhenReader(),
      getPluginCommands: () => [],
      getDeclaredCommands: () => [],
      getEditorCommands: () => [],
      getHandlerDeclarations: () => [
        {
          pluginId: PAGES,
          command: 'app:file.createText',
          title: 'Extract text layer',
          when: "target.contentType == 'application/pdf'",
          hasItems: false,
        },
        {
          pluginId: SCANS,
          command: 'app:file.createText',
          title: 'Recognize text',
          when: "target.contentType == 'application/pdf'",
          hasItems: false,
        },
      ],
      getWhenValues: (target) => getTargetValues(target, { pdf: 'application/pdf' }),
    })
    const registry = createSlotRegistry({
      getEntries: () =>
        readSlotEntries([
          {
            pluginId: PAGES,
            slot: 'fileTree.context',
            command: 'app:file.createText',
            group: '3-text',
            order: 10,
          },
          {
            pluginId: SCANS,
            slot: 'fileTree.context',
            command: 'app:file.createText',
            group: '3-text',
            order: 10,
          },
        ])?.filter((one) => one.slot === 'fileTree.context'),
      findCommand,
      findView: () => undefined,
    })

    expect(registry.getEntries('fileTree.context', PDF).map(describeItem)).toStrictEqual([
      'app:file.createText',
    ])
  })
})
