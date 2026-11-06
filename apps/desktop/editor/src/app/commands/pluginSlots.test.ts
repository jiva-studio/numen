/** The slot entries of the window: those the plugins declare and those of the application, and how their commands are drawn. */
import { describe, expect, it } from 'vitest'
import { getTargetValues, type CommandTarget } from '@numen/plugin-host'
import type { DeclaredCommand } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/icons'
import { APP_SLOT_ENTRIES } from './appPlacements'
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
      },
    ])
  })

  it('place the commands of the window over a row in the menu of the file tree', () => {
    expect(APP_SLOT_ENTRIES.filter((one) => one.slot === 'fileTree.context')).toStrictEqual([
      {
        pluginId: 'app',
        slot: 'fileTree.context',
        command: 'app:note.open',
        group: '0-open',
        order: 10,
      },
      {
        pluginId: 'app',
        slot: 'fileTree.context',
        command: 'app:note.copyPath',
        group: '2-edit',
        order: 20,
        when: '!target.isRoot && !target.isMultiple',
      },
      { pluginId: 'app', slot: 'fileTree.context', command: 'app:note.trash', group: '9-remove' },
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
    getPluginCommands: () => [
      { id: `${TIDE}:tide.live`, title: 'Live tide', icon: 'waves', shortcut: 'mod+t' },
    ],
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

  it('is drawn with the title, icon and shortcut of the command', () => {
    expect(find(`${TIDE}:tide.live`, undefined)).toStrictEqual({
      title: 'Live tide',
      icon: getNamedIcon('waves'),
      shortcut: 'mod+t',
      hasItems: false,
    })
    expect(find(`${TIDE}:tide.open`, undefined)).toStrictEqual({
      title: 'Open tide',
      icon: getNamedIcon('waves'),
      shortcut: '',
      hasItems: false,
    })
    expect(find('app:settings.open', undefined)).toStrictEqual({
      title: 'Open settings',
      icon: getNamedIcon('settings'),
      shortcut: '',
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
