/**
 * The window composed in steps, each made from the ones before: the base (the tabs, the vault, the
 * commands of the features), the plugins, the palettes, the keys, the settings page and the start.
 */
import { loadFrameRuntime, useWindowFrame } from '@numen/workbench'
import { createWindowBase } from './createWindowBase'
import { createWindowKeys } from './createWindowKeys'
import { createWindowPalettes } from './createWindowPalettes'
import { createWindowPlugins } from './createWindowPlugins'
import { createWindowStart } from './createWindowStart'
import { registerSettingsPage } from './registerSettingsPage'

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const base = createWindowBase()
  const { messages, settings, commands, tabs, tabParts, vault } = base
  const frame = useWindowFrame(loadFrameRuntime)
  const plugins = createWindowPlugins({
    log: messages.log,
    rest: settings.rest,
    frame,
    status: base.status,
    commands,
    windowSettings: settings.window,
    tabs,
    vault,
  })
  const palettes = createWindowPalettes({
    rest: settings.rest,
    writeMessage: messages.writeMessage,
    openSettings: base.settingsRequests.emit,
    commands,
    windowSettings: settings.window,
    tabs,
    vault,
    plugins,
  })
  createWindowKeys({
    windowTabs: tabs.windowTabs,
    getTarget: commands.getTarget,
    palette: palettes.palette,
    pluginCommands: palettes.pluginPalette.plugin.pluginCommands,
  })
  registerSettingsPage({
    rest: settings.rest,
    connections: settings.connections,
    windowSettings: settings.window,
    log: messages.log,
    windowTabs: tabs.windowTabs,
    plugins: {
      host: plugins.pluginHost,
      health: vault.window.pluginHealth,
      commands: plugins.hostPort.commands,
    },
    frame,
    settingsRequests: base.settingsRequests,
  })
  const { views, tabMenu, bootstrap } = createWindowStart({
    writeMessage: messages.writeMessage,
    settings,
    frame,
    tabs,
    tabParts,
    vault,
    plugins,
    palettes,
  })
  const { getTarget } = commands
  return { tabs, tabParts, vault, getTarget, palettes, views, tabMenu, bootstrap }
}

/** What the component that draws the window is given. */
export type WindowParts = ReturnType<typeof createWindow>
