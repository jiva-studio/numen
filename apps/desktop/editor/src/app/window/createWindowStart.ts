/**
 * The keys of the window and its start: the parts that connect the others to what the person does.
 */
import type { PluginLoader } from '@numen/plugin-loader'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { useVaults } from '@/entities/vault'
import { createEditorTokens } from '@/features/editor-tokens'
import { closeWindow } from '@/entities/window'
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
import type { WindowApi } from './createWindowApi'

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
    listPlugins: () => rest.state.plugins.value,
    waitForDeclaredCommands: () => declaredCommands.waitForFirstList(),
    writeMessage: deps.writeMessage,
  })
}

type WindowBase = ReturnType<typeof createWindowBase>

/** What the start of the layout of the tabs reads. */
export interface LayoutStarterDeps {
  readonly api: WindowApi['tabs']
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>
  readonly tabs: WindowBase['tabs']
  readonly tabParts: WindowBase['tabParts']
  readonly closeWhenEmpty: (() => Promise<void>) | undefined
  readonly pluginLoader: PluginLoader
  readonly writeMessage: MessageWriter
}

function createLayoutStarter(deps: LayoutStarterDeps) {
  const { tabs, tabParts } = deps
  return createLayoutStart({
    vaults: deps.vaults,
    windowTabs: tabs.windowTabs,
    pluginLoader: deps.pluginLoader,
    writeMessage: deps.writeMessage,
    layout: tabParts.layout,
    saved: {
      readDetachedTab: (tabId) => deps.api.tabMove.readDetachedTab(tabs.windowId, tabId),
      readOpenTabs: deps.api.savedTabs.readOpenTabs,
    },
    openDetachedTab: tabs.openMovedTab,
    closeWhenEmpty: deps.closeWhenEmpty,
  })
}

/** What the start of the window reads of the foundation. */
type StartFoundation = Pick<
  WindowBase['foundation'],
  'api' | 'log' | 'rest' | 'frame' | 'connections' | 'settingsRequests'
>

/** What the start of the window reads of the base, the plugins and the palettes. */
export interface WindowStartDeps {
  readonly foundation: StartFoundation
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
  const { api, log, rest, frame } = foundation
  const writeMessage = log.getWriter('command')
  const { windowTabs } = tabs
  const { pluginHost, pluginLoader, declaredCommands } = plugins
  const { editing, window, vaults } = vault
  registerSettingsPage({
    foundation,
    windowSettings,
    windowTabs,
    plugins: {
      host: pluginHost,
      health: window.pluginHealth,
      commands: plugins.hostPort.commands,
      getSlotGroups: plugins.slotGroups,
    },
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
      tabLinks: tabs.tabLinks,
      moves: tabParts.moves,
      pluginHost,
      windowCommands: palettes.windowCommands,
      itemLists: vault.itemLists,
      writeMessage,
    }),
    bootstrap: createWindowBootstrap({
      vaults,
      settings: {
        start: () => windowSettings.start(),
        close: () => windowSettings.close(),
        editorTokens: createEditorTokens(rest),
      },
      window,
      editing,
      tabLinkSession: tabs.linkSession,
      writeMessage,
      layout: createLayoutStarter({
        api: api.tabs,
        vaults,
        tabs,
        tabParts,
        pluginLoader,
        closeWhenEmpty: tabParts.isSecondary
          ? () => closeWindow(foundation.api.window.frameRuntime)
          : undefined,
        writeMessage,
      }),
      plugins: createPluginStarter({ pluginLoader, rest, declaredCommands, writeMessage }),
    }),
  }
}
