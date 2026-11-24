import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createCommandRegistry, type CommandTarget } from '@/shared/model/command'
import { createAppCommands } from '@/testing/appCommands'
import { runSupport } from '../palette/runSupport'
import { createWindowPalette } from '../window/createWindowPalette'
import { createCommandDeps } from './createCommandDeps'
import { createPaletteSwitch } from './createPaletteSwitch'
import { createSearchRequests } from './createSearchRequests'

const setup = () => {
  const commands = {
    palette: {
      open: ref(false),
      setOpen: (isOpen: boolean) => (commands.palette.open.value = isOpen),
    },
  }
  const search = { open: ref(false), setOpen: (isOpen: boolean) => (search.open.value = isOpen) }
  return { commands, search, palettes: createPaletteSwitch(commands as never, search as never) }
}

describe('the two palettes of the window', () => {
  it('open the search over the commands, and close it when it is open', () => {
    const { commands, search, palettes } = setup()
    expect(palettes.isOpen()).toBe(false)
    commands.palette.open.value = true
    expect(palettes.isOpen()).toBe(true)

    palettes.toggleSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, true])
    expect(palettes.isOpen()).toBe(true)

    palettes.closeSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, false])

    palettes.toggleSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, true])

    palettes.toggleSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, false])
  })

  it('open the commands over the search, and close them when they are open', () => {
    const { commands, search, palettes } = setup()
    search.open.value = true

    palettes.toggleCommands()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([true, false])

    palettes.toggleCommands()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, false])
  })

  it('open the search on a request made before they existed', () => {
    const { commands, search, palettes } = setup()
    commands.palette.open.value = true
    const requests = createSearchRequests()
    requests.request()

    requests.connect(palettes)
    requests.request()

    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, true])
  })

  it('connects search requests, resets lists when closed, and loads vaults in createWindowPalette', async () => {
    const resetWhenClosed = vi.fn()
    const getVaultList = vi.fn().mockResolvedValue({
      vaults: [{ id: 'v1', name: 'One', path: '/v1', isMissing: false }],
      showing: 'v1',
    })
    const searchRequests = createSearchRequests()
    const target: CommandTarget = {
      tab: '',
      kind: null,
      path: '',
      title: '',
      file: '',
      vault: { id: 'v1', name: 'One' },
      isReady: true,
    }
    const windowPalette = createWindowPalette({
      searchCore: { searchNames: async () => [], search: async () => [] },
      commandDeps: createCommandDeps({
        windowTabs: {} as never,
        reached: {} as never,
        settingPalette: { choose: async () => undefined },
        runs: runSupport(),
        registry: createCommandRegistry(),
        writeMessage: vi.fn(),
        navigation: { search: vi.fn() },
      }),
      getTarget: () => target,
      knows: { getTitle: () => '', getTabAt: () => null },
      plugin: {
        lists: {
          lists: { getStepGroups: () => [], applyItemPreview: vi.fn() },
          removeEmptyCommands: (cmds) => cmds,
          resetWhenClosed,
        },
        getCommands: createAppCommands,
        pluginCommands: { isPluginCommand: () => false, run: async () => undefined },
      },
      searchRequests,
      searchOptions: {},
      getVaultList,
    })

    expect(resetWhenClosed).toHaveBeenCalledWith(windowPalette.commands.palette.open)
    searchRequests.request()
    expect(windowPalette.search.open.value).toBe(true)

    windowPalette.commands.palette.startCommand('app:vault.open', target)
    await vi.waitFor(() => expect(getVaultList).toHaveBeenCalledOnce())
  })
})
