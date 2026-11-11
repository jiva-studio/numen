/** The slot entries of the window: those the plugins declare and those of the application, and how their commands are drawn. */
import { describe, expect, it } from 'vitest'
import { FileText } from '@lucide/vue'
import { createWhenReader, getTargetValues, type CommandTarget } from '@numen/plugin-loader'
import type { DeclaredCommand } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/ui/icons'
import { APP_SLOT_ENTRIES } from '../../lib/appPlacements'
import { createSlotCommandFinder, readSlotEntries } from './pluginSlots'

const TIDE = 'org.example.tide'

const declared = (one: Partial<DeclaredCommand> & { id: string }): DeclaredCommand => ({
  pluginId: TIDE,
  title: one.id,
  arguments: [],
  ...one,
})

const NOTE: CommandTarget = {
  path: 'notes/Bees.md',
  title: 'Bees',
  isFolder: false,
  others: [],
  tab: '',
}
const FOLDER: CommandTarget = { path: 'notes', title: 'notes', isFolder: true, others: [], tab: '' }

describe('the slot entries of the window', () => {
  it('are none until the plugins have been listed, then theirs and those of the application', () => {
    const tide = { pluginId: TIDE, slot: 'titleBar.commands', command: 'tide.open' }

    expect(readSlotEntries(undefined)).toBeUndefined()
    expect(readSlotEntries([tide])).toStrictEqual([tide, ...APP_SLOT_ENTRIES])
  })

  it('carry the icon component of a submenu', () => {
    const [submenu] =
      readSlotEntries([
        {
          pluginId: TIDE,
          slot: 'fileTree.context',
          submenu: 'tide.new',
          title: 'New',
          iconName: 'plus',
        },
      ]) ?? []

    expect(submenu).toMatchObject({ submenu: 'tide.new', title: 'New', icon: getNamedIcon('plus') })
  })

  it('place the settings button of the application last in the title bar', () => {
    expect(APP_SLOT_ENTRIES.filter((one) => one.slot === 'titleBar.commands')).toStrictEqual([
      {
        pluginId: 'app',
        slot: 'titleBar.commands',
        command: 'app:settings.open',
        group: 'settings',
        when: 'window.isVaultOpen',
      },
    ])
  })

  it('place the commands of the window over a row in the menu of the file tree', () => {
    expect(APP_SLOT_ENTRIES.filter((one) => one.slot === 'fileTree.context')).toStrictEqual([
      {
        pluginId: 'app',
        slot: 'fileTree.context',
        command: 'app:file.open',
        group: '0-open',
        order: 10,
      },
      {
        pluginId: 'app',
        slot: 'fileTree.context',
        command: 'app:file.copyPath',
        group: '2-edit',
        order: 20,
        when: '!target.isRoot && !target.isMultiple',
      },
      {
        pluginId: 'app',
        slot: 'fileTree.context',
        command: 'app:version.delete',
        group: '3-text',
        order: 90,
      },
      { pluginId: 'app', slot: 'fileTree.context', command: 'app:file.remove', group: '9-remove' },
    ])
  })

  it('place the commands of the window over a tab in the menu of a tab', () => {
    expect(APP_SLOT_ENTRIES.filter((one) => one.slot === 'tabStrip.context')).toStrictEqual([
      {
        pluginId: 'app',
        slot: 'tabStrip.context',
        command: 'app:tab.moveToWindow',
        group: '0-window',
        order: 10,
      },
      {
        pluginId: 'app',
        slot: 'tabStrip.context',
        command: 'app:tab.detach',
        group: '0-window',
        order: 20,
      },
      {
        pluginId: 'app',
        slot: 'tabStrip.context',
        command: 'app:tab.link',
        group: '1-link',
        order: 10,
      },
      {
        pluginId: 'app',
        slot: 'tabStrip.context',
        command: 'app:tab.unlink',
        group: '1-link',
        order: 20,
      },
    ])
  })
})

describe('a command of a slot entry', () => {
  const find = createSlotCommandFinder({
    whenReader: createWhenReader(),
    getPluginCommands: () => [{ id: `${TIDE}:tide.live`, title: 'Live tide', icon: 'waves' }],
    getDeclaredCommands: () => [
      declared({ id: 'tide.open', title: 'Open tide', icon: 'waves' }),
      declared({ id: 'tide.note', title: 'Tide note', when: 'target.isNote' }),
      declared({
        id: 'tide.chart',
        title: 'Chart',
        when: "target.contentType == 'text/csv'",
      }),
    ],
    getEditorCommands: () => [
      { id: 'app:settings.open', text: 'Open settings', iconName: 'settings' },
    ],
    getWhenValues: (target) => getTargetValues(target, { csv: 'text/csv', md: 'text/markdown' }),
  })

  it('is drawn with the title and icon of the command', () => {
    expect(find(`${TIDE}:tide.live`, undefined)).toStrictEqual({
      title: 'Live tide',
      icon: getNamedIcon('waves'),
      hasItems: false,
    })
    expect(find(`${TIDE}:tide.open`, undefined)).toStrictEqual({
      title: 'Open tide',
      icon: getNamedIcon('waves'),
      hasItems: false,
    })
    expect(find('app:settings.open', undefined)).toStrictEqual({
      title: 'Open settings',
      icon: getNamedIcon('settings'),
      hasItems: false,
    })
  })

  it('is not available where no command of the id is listed', () => {
    expect(find(`${TIDE}:tide.gone`, undefined)).toBeUndefined()
    expect(find('app:gone.away', undefined)).toBeUndefined()
  })

  it('over a note is available only over a target that is no folder', () => {
    expect(find(`${TIDE}:tide.note`, undefined)).toBeUndefined()
    expect(find(`${TIDE}:tide.note`, FOLDER)).toBeUndefined()
    expect(find(`${TIDE}:tide.note`, NOTE)).toMatchObject({ title: 'Tide note' })
  })

  it('over files is available only over a file of its content types', () => {
    expect(find(`${TIDE}:tide.chart`, undefined)).toBeUndefined()
    expect(find(`${TIDE}:tide.chart`, NOTE)).toBeUndefined()
    expect(find(`${TIDE}:tide.chart`, { ...NOTE, path: 'data/tides.csv' })).toMatchObject({
      title: 'Chart',
    })
  })
})

describe('a shared command in a slot', () => {
  const find = createSlotCommandFinder({
    whenReader: createWhenReader(),
    getPluginCommands: () => [],
    getDeclaredCommands: () => [],
    getEditorCommands: () => [],
    getHandlerDeclarations: () => [
      { pluginId: TIDE, command: 'app:file.createText', title: 'Extract text', hasItems: false },
    ],
    getWhenValues: (target) => getTargetValues(target, {}),
  })

  it('is drawn under its own title as a command with items, its handlers', () => {
    expect(find('app:file.createText', NOTE)).toStrictEqual({
      title: 'Create text',
      icon: FileText,
      hasItems: true,
    })
  })

  it('is not available while no enabled plugin handles it', () => {
    expect(find('app:link.deleteMedia', NOTE)).toBeUndefined()
  })
})

describe('a shared command over what the window knows of the target', () => {
  const PDF = 'application/pdf'
  const DECK: CommandTarget = { ...NOTE, path: 'cards/Bees.md', noteType: 'deck' }
  const LINK: CommandTarget = { ...NOTE, path: 'links/Bees.md', noteType: 'link' }
  const PAPER: CommandTarget = { ...NOTE, path: 'papers/Bees.pdf' }
  const find = createSlotCommandFinder({
    whenReader: createWhenReader(),
    getPluginCommands: () => [],
    getDeclaredCommands: () => [],
    getEditorCommands: () => [],
    getHandlerDeclarations: () => [
      {
        pluginId: TIDE,
        command: 'app:file.createText',
        title: 'Extract text',
        when: `target.contentType == '${PDF}'`,
        hasItems: false,
      },
      {
        pluginId: TIDE,
        command: 'app:link.deleteMedia',
        title: 'Delete media',
        when: "target.noteType == 'link' && has(target.properties.media)",
        hasItems: false,
      },
      {
        pluginId: TIDE,
        command: 'app:link.downloadMedia',
        title: 'Download media',
        when: "!target.versions.exists(v, v.kind == 'captions')",
        hasItems: true,
      },
    ],
    getWhenValues: (target) =>
      getTargetValues(target, { pdf: PDF, md: 'text/markdown' }, { type: target?.noteType }),
  })

  it('is not drawn where every handler is false over the note type and the content type', () => {
    expect(find('app:file.createText', DECK)).toBeUndefined()
    expect(find('app:link.deleteMedia', DECK)).toBeUndefined()
  })

  it('is drawn where a handler is true over what is known', () => {
    expect(find('app:file.createText', PAPER)).toMatchObject({ title: 'Create text' })
  })

  it('is drawn where a handler depends on the properties of the note, which are not read yet', () => {
    expect(find('app:link.deleteMedia', LINK)).toMatchObject({ title: 'Delete media' })
  })

  it('is drawn where a handler depends on the versions of the file', () => {
    expect(find('app:link.downloadMedia', DECK)).toMatchObject({ title: 'Download media' })
  })

  it('is drawn where the place has not read the type of the note', () => {
    expect(find('app:link.deleteMedia', { ...NOTE, path: 'links/Bees.md' })).toMatchObject({
      title: 'Delete media',
    })
  })
})
