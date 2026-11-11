/**
 * The keys of the window, the settings page and the start: the parts that connect the others to
 * what the person does.
 */
import { readDetachedTab } from '@/features/tab-move'
import { createLayoutStart, createPluginStart, createWindowBootstrap } from '@/features/app-start'
import { createAppKeyRunner, createKeyHandler, syncKeybindings, useAppHotkeys } from '@/features/hotkeys'
import type { WindowFoundation } from './createWindowFoundation'
import type { WindowFeatures } from './createWindowFeatures'
import type { WindowPalettes } from './createWindowPalettes'
import type { WindowPlugins } from './createWindowPlugins'
import type { WindowVault } from './createWindowVault'

/** What the keys and the start of the window use of the other parts of it. */
export interface WindowStartDeps {
  readonly foundation: Pick<WindowFoundation, 'rest' | 'writeMessage'>
  readonly parts: Pick<WindowVault, 'tabsPart' | 'vault' | 'windowSettings' | 'tabs'>
  readonly getTarget: WindowFeatures['getTarget']
  readonly plugins: Pick<WindowPlugins, 'support' | 'declaredCommands'>
  readonly palettes: Pick<WindowPalettes, 'paletteSwitch' | 'pluginCommands' | 'commandRuns'>
}

/** Puts the keys of the window to the commands, and starts the window when it mounts. */
export function createWindowStart(deps: WindowStartDeps) {
  const { rest, writeMessage } = deps.foundation
  const { windowTabs, windowId } = deps.parts.tabsPart
  const { vault, windowSettings, tabs } = deps.parts
  const { getTarget } = deps
  const { pluginHost, pluginLoader } = deps.plugins.support
  const { declaredCommands } = deps.plugins
  const { paletteSwitch, pluginCommands, commandRuns } = deps.palettes

  syncKeybindings(
    pluginHost.keybindings,
    declaredCommands.contributions.keybindings,
    rest.state.plugins,
  )
  useAppHotkeys(
    createKeyHandler({
      windowTabs,
      handleShortcut: (event, runApp) => pluginCommands.handleShortcut(event, getTarget, runApp),
      runApp: createAppKeyRunner({
        toggleSearch: paletteSwitch.toggleSearch,
        toggleCommands: paletteSwitch.toggleCommands,
        runCommand: commandRuns.runCommand,
        runWithValues: commandRuns.runWithValues,
        getTarget,
      }),
    }),
  )
  return createWindowBootstrap({
    vaults: vault.vaults,
    settings: windowSettings,
    window: vault.window,
    editing: vault.editing,
    tabLinkSession: tabs.linkSession,
    writeMessage,
    plugins: createPluginStart({
      pluginLoader,
      getPlugins: () => rest.state.plugins.value,
      waitForDeclaredCommands: () => declaredCommands.waitForFirstList(),
      writeMessage,
    }),
    layout: createLayoutStart({
      vaults: vault.vaults,
      windowTabs,
      pluginLoader,
      writeMessage,
      layout: tabs.layout,
      isSecondary: tabs.isSecondary,
      windowId,
      readDetachedTab,
      openDetachedTab: tabs.openMovedTab,
    }),
  })
}
