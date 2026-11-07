/**
 * The commands the window gives its plugins: the ids of its own and of the plugins it runs, and the
 * run of one through the runner the palette and the shortcuts use.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import type { CommandTarget as RunTarget } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import { createWindowCommands } from './windowCommands'

const WINDOWS = [{ id: 'w2', title: 'Second window' }]

const FRONT = {
  tab: 'plex:1',
  kind: 'plex',
  path: 'Front.md',
  title: 'Front',
} as unknown as CommandTarget

function setup() {
  const pluginIds = shallowRef<readonly string[]>(['org.example.tide:chart.open'])
  const runCommand = vi.fn<(id: string, target: CommandTarget) => void>()
  const runWithArgument = vi.fn(
    async (_id: string, _target: RunTarget, _argument: string) => undefined,
  )
  const listItems = vi.fn(async (id: string, _target: RunTarget) =>
    id === 'org.example.tide:chart.move' ? WINDOWS : [],
  )
  const commands = createWindowCommands({
    getAppIds: () => ['app:vault.create', 'app:settings.open'],
    getPluginIds: () => pluginIds.value,
    runCommand,
    listItems,
    runWithArgument,
    getTarget: () => FRONT,
  })
  return { commands, pluginIds, runCommand, runWithArgument, listItems }
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

  it('run a command with no target over the target of the tab in front', async () => {
    const { commands, runCommand } = setup()

    await commands.run('app:vault.create')

    expect(runCommand).toHaveBeenCalledExactlyOnceWith('app:vault.create', FRONT)
  })

  it('run a command over the target given, its file and the others selected with it included, and mark the target named', async () => {
    const { commands, runCommand } = setup()
    const target: RunTarget = {
      path: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: ['reef/Kelp.md'],
      tab: '',
    }

    await commands.run('app:note.trash', target)

    expect(runCommand).toHaveBeenCalledExactlyOnceWith('app:note.trash', {
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
    const { commands, runCommand } = setup()
    const target: RunTarget = {
      path: 'reef/Coral.md',
      title: 'Coral',
      isFolder: false,
      others: [],
      tab: 'note:7',
    }

    await commands.run('app:tab.close', target)

    expect(runCommand.mock.calls[0]?.[1].tab).toBe('note:7')
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

  it('run a command with the item picked as its argument', async () => {
    const { commands, runCommand, runWithArgument } = setup()

    await commands.run('org.example.tide:chart.move', undefined, 'w2')

    expect(runCommand).not.toHaveBeenCalled()
    expect(runWithArgument.mock.calls[0]?.[0]).toBe('org.example.tide:chart.move')
    expect(runWithArgument.mock.calls[0]?.[2]).toBe('w2')
  })
})
