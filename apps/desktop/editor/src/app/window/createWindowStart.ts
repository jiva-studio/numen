/**
 * The keys of the window and its start: the parts that connect the others to what the person does.
 */
import type { PluginLoader } from '@numen/plugin-loader'
import type { SettingsStore } from '@/entities/settings'
import type { createWindowLayout, WindowTabs } from '@/entities/tab'
import type { useVaults } from '@/entities/vault'
import { readDetachedTab } from '@/features/tab-move'
import { createLayoutStart, createPluginStart, createWindowBootstrap } from '@/features/app-start'
import type { createWindowPalette } from '@/features/command-palette'
import { createAppKeyRunner, createKeyHandler, useAppHotkeys } from '@/features/hotkeys'
import type { createPluginCommandPort, DeclaredCommands } from '@/features/plugin-commands'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { createWindowBase } from './createWindowBase'
import { createWindowViews } from './createPluginHost'
import type { createWindowPalettes } from './createWindowPalettes'
import type { createWindowPlugins } from './createWindowPlugins'
import { createWindowTabMenu } from './createWindowTabMenu'
import { registerSettingsPage } from './registerSettingsPage'

/** What the keys of the window reach. */
interface WindowKeysDeps {
  readonly windowTabs: Pick<WindowTabs, 'panes'>
  readonly getTarget: () => CommandTarget
  readonly palette: Pick<ReturnType<typeof createWindowPalette>, 'paletteSwitch' | 'runs'>
  readonly pluginCommands: ReturnType<typeof createPluginCommandPort>
}

function createWindowKeys(deps: WindowKeysDeps): void {
  const { windowTabs, getTarget, palette, pluginCommands } = deps
  useAppHotkeys(
    createKeyHandler({
      windowTabs,
      getTarget,
      handleShortcut: pluginCommands.handleShortcut,
      runApp: createAppKeyRunner({
        toggleSearch: palette.paletteSwitch.toggleSearch,
        toggleCommands: palette.paletteSwitch.toggleCommands,
        runCommand: palette.runs.runCommand,
        runWithValues: palette.runs.runWithValues,
        getTarget,
      }),
    }),
  )
}

/** What the start of the plugins of the window reads. */
export interface PluginStarterDeps {
  readonly pluginLoader: PluginLoader
  readonly rest: Pick<SettingsStore, 'state'>
  readonly declaredCommands: Pick<DeclaredCommands, 'waitForFirstList'>
  readonly writeMessage: MessageWriter
}

function createPluginStarter(deps: PluginStarterDeps) {
  const { rest, declaredCommands } = deps
  return createPluginStart({
    pluginLoader: deps.pluginLoader,
    getPlugins: () => rest.state.plugins.value,
    waitForDeclaredCommands: () => declaredCommands.waitForFirstList(),
    writeMessage: deps.writeMessage,
  })
}

/** What the start of the layout of the tabs reads. */
export interface LayoutStarterDeps {
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>
  readonly windowTabs: WindowTabs
  readonly pluginLoader: PluginLoader
  readonly layout: ReturnType<typeof createWindowLayout>
  readonly isSecondary: boolean
  readonly windowId: string
  readonly openMovedTab: Parameters<typeof createLayoutStart>[0]['openDetachedTab']
  readonly writeMessage: MessageWriter
}

function createLayoutStarter(deps: LayoutStarterDeps) {
  return createLayoutStart({
    vaults: deps.vaults,
    windowTabs: deps.windowTabs,
    pluginLoader: deps.pluginLoader,
    writeMessage: deps.writeMessage,
    layout: deps.layout,
    isSecondary: deps.isSecondary,
    windowId: deps.windowId,
    readDetachedTab,
    openDetachedTab: deps.openMovedTab,
  })
}

type WindowBase = ReturnType<typeof createWindowBase>

/** What the start of the window reads of the base, the plugins and the palettes. */
export interface WindowStartDeps {
  readonly foundation: WindowBase['foundation']
  readonly tabs: WindowBase['tabs']
  readonly tabParts: WindowBase['tabParts']
  readonly vault: WindowBase['vault']
  readonly windowSettings: WindowBase['windowSettings']
  readonly getTarget: WindowBase['getTarget']
  readonly plugins: ReturnType<typeof createWindowPlugins>
  readonly palettes: ReturnType<typeof createWindowPalettes>
}

export function createWindowStart(deps: WindowStartDeps) {
  const { foundation, tabs, tabParts, vault, windowSettings, getTarget, plugins, palettes } = deps
  const { log, rest, connections, frame, writeMessage, settingsRequests } = foundation
  const { windowTabs } = tabs
  const { pluginHost, pluginLoader, declaredCommands } = plugins
  const { editing, window, vaults } = vault
  registerSettingsPage({
    rest,
    connections,
    windowSettings,
    log,
    windowTabs,
    pluginHost,
    frame,
    settingsRequests,
    health: window.pluginHealth,
    commands: plugins.hostPort.commands,
  })
  createWindowKeys({
    windowTabs,
    getTarget,
    palette: palettes.palette,
    pluginCommands: palettes.pluginPalette.plugin.pluginCommands,
  })
  return {
    views: createWindowViews({ pluginHost, frame }),
    tabMenu: createWindowTabMenu({
      windowTabs,
      moves: tabParts.moves,
      pluginHost,
      windowCommands: palettes.windowCommands,
      itemLists: vault.itemLists,
      writeMessage,
    }),
    bootstrap: createWindowBootstrap({
      vaults,
      settings: windowSettings,
      window,
      editing,
      tabLinkSession: tabs.linkSession,
      writeMessage,
      layout: createLayoutStarter({
        vaults,
        windowTabs,
        pluginLoader,
        layout: tabParts.layout,
        isSecondary: tabParts.isSecondary,
        windowId: tabs.windowId,
        openMovedTab: tabs.openMovedTab,
        writeMessage,
      }),
      plugins: createPluginStarter({ pluginLoader, rest, declaredCommands, writeMessage }),
    }),
  }
}
