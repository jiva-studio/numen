/**
 * The commands the window gives its plugins: the ids of its own and of the plugins it runs, and the
 * run of one through the runner the palette and the shortcuts use.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import type { CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'
import { createWindowCommands } from './windowCommands'

const WINDOWS = [{ id: 'w2', title: 'Second window' }]

const FOLDER: RunTarget = { path: '', title: '', isFolder: true, others: [], tab: '' }

const FRONT = {
  tab: 'plex:1',
  kind: 'plex',
  path: 'Front.md',
  title: 'Front',
} as unknown as CommandTarget

function setup() {
  const pluginIds = shallowRef<readonly string[]>(['org.example.tide:chart.open'])
  const runCommandToEnd = vi.fn(
    async (_id: string, _target: CommandTarget, _values?: Readonly<Record<string, string>>) => '',
  )
  const runWithItem = vi.fn(
    async (
      _id: string,
      _target: RunTarget,
      _argument: string,
      _values?: Readonly<Record<string, string>>,
    ) => undefined,
  )
  const listItems = vi.fn(async (id: string, _target: RunTarget) =>
    id === 'org.example.tide:chart.move' ? WINDOWS : [],
  )
  const commands = createWindowCommands({
    getAppIds: () => ['app:vault.create', 'app:settings.open'],
    getPluginIds: () => pluginIds.value,
    runCommandToEnd,
    listItems,
    runWithItem,
    getTarget: () => FRONT,
  })
  return { commands, pluginIds, runCommandToEnd, runWithItem, listItems }
}

describe('the commands of the window', () => {
  it('are the commands of the window and then those of its plugins', () => {
    expect(setup().commands.getIds()).toStrictEqual([
      'app:vault.create',
      'app:settings.open',
      'org.example.tide:chart.open',
    ])
  })

  it('call a watcher at once and again when a plugin starts or stops, until it is disposed', async () => {
    const { commands, pluginIds } = setup()
    const calls: (readonly string[])[] = []
    const watch = commands.watch((ids) => calls.push(ids))

    pluginIds.value = []
    await nextTick()
    watch.dispose()
    pluginIds.value = ['org.example.reef:map.open']
    await nextTick()

    expect(calls.map((ids) => ids.length)).toStrictEqual([3, 2])
  })

  it('run a command with no target, as a bar does, over the file of the tab in front, named as a menu names its target', async () => {
    const { commands, runCommandToEnd } = setup()

    await commands.run('org.example.tide:chart.open')

    expect(runCommandToEnd).toHaveBeenCalledExactlyOnceWith('org.example.tide:chart.open', {
      ...FRONT,
      isTargetNamed: true,
    })
  })

  it('run a command over the target given, its file and the others selected with it included, and mark the target named', async () => {
    const { commands, runCommandToEnd } = setup()
    const target: RunTarget = {
      path: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: ['reef/Kelp.md'],
      tab: '',
    }

    await commands.run('app:file.remove', target)

    expect(runCommandToEnd).toHaveBeenCalledExactlyOnceWith('app:file.remove', {
      ...FRONT,
      path: 'reef/Coral.md',
      file: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: ['reef/Kelp.md'],
      isTargetNamed: true,
    })
  })

  it('run a command over the tab a tab menu names, in place of the tab in front', async () => {
    const { commands, runCommandToEnd } = setup()
    const target: RunTarget = {
      path: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: [],
      tab: 'note:7',
    }

    await commands.run('app:tab.close', target)

    expect(runCommandToEnd.mock.calls[0]?.[1].tab).toBe('note:7')
  })

  it('give the items a command offers over the target, over the tab in front where none is named', async () => {
    const { commands, listItems } = setup()

    expect(await commands.listItems('org.example.tide:chart.move')).toStrictEqual(WINDOWS)
    expect(await commands.listItems('app:vault.create')).toStrictEqual([])
    expect(listItems.mock.calls[0]?.[1]).toMatchObject({
      path: 'Front.md',
      title: 'Front',
      tab: 'plex:1',
    })
  })

  it('resolve with the path of the note the run made, and with an empty path for a run with an item', async () => {
    const { commands, runCommandToEnd } = setup()
    runCommandToEnd.mockResolvedValueOnce('Untitled note 1.md')

    expect(await commands.run('app:note.create', FOLDER)).toBe('Untitled note 1.md')
    expect(await commands.run('org.example.tide:chart.move', undefined, 'w2')).toBe('')
  })

  it('run a command with the item picked as its itemId', async () => {
    const { commands, runCommandToEnd, runWithItem } = setup()

    await commands.run('org.example.tide:chart.move', undefined, 'w2')

    expect(runCommandToEnd).not.toHaveBeenCalled()
    expect(runWithItem.mock.calls[0]?.[0]).toBe('org.example.tide:chart.move')
    expect(runWithItem.mock.calls[0]?.[2]).toBe('w2')
  })

  it('run a command to its end, and reject with the error of the run', async () => {
    const { commands, runCommandToEnd } = setup()
    let isEnded = false
    runCommandToEnd.mockImplementationOnce(async () => {
      await Promise.resolve()
      isEnded = true
      throw new Error('the disk is full')
    })

    await expect(commands.run('org.example.tide:chart.open')).rejects.toThrow('the disk is full')
    expect(isEnded).toBe(true)
  })
})

describe('the run of a command with values', () => {
  it('gives the values to the run of a command without items', async () => {
    const { commands, runCommandToEnd } = setup()

    await commands.run('app:note.show', undefined, undefined, { where: 'plex' })

    expect(runCommandToEnd).toHaveBeenCalledWith('app:note.show', expect.anything(), {
      where: 'plex',
    })
  })

  it('gives the values to the run of a command with an item', async () => {
    const { commands, runWithItem } = setup()

    await commands.run('org.example.tide:chart.move', FOLDER, 'w2', { where: 'plex' })

    expect(runWithItem).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      expect.anything(),
      'w2',
      { where: 'plex' },
    )
  })
})
