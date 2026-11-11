/**
 * The window composed: the services that read the vault, the commands, the kinds of tab and the
 * plugins, made in the order each needs the one before.
 */
import { createFileMakers, createNoteKindReader } from '@/features/plugin-commands'
import { createSettingsPage } from '@/pages/settings'
import { createHostConnection } from './createHostConnection'
import { createWindowFeatures } from './createWindowFeatures'
import { createWindowFoundation } from './createWindowFoundation'
import { createWindowPalettes } from './createWindowPalettes'
import { createWindowPlugins } from './createWindowPlugins'
import { createWindowStart } from './createWindowStart'
import { createWindowVault } from './createWindowVault'
import { getPluginViews } from './getPluginViews'

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const foundation = createWindowFoundation()
  const { frame, rest, log, settingsRequests } = foundation
  const parts = createWindowVault(foundation)
  const { windowTabs, tabOpeners } = parts.tabsPart
  const features = createWindowFeatures({ foundation, parts })
  const plugins = createWindowPlugins({ foundation, parts, features })
  const { pluginHost } = plugins.support
  const palettes = createWindowPalettes({ foundation, parts, features, plugins })

  const settingsPage = createSettingsPage({
    rest,
    appearance: parts.windowSettings.appearance,
    log,
    windowTabs,
    getSettingsViews: (pluginId) => pluginHost.registered.getRegisteredSettingsViews(pluginId),
    getPlatform: () => frame.platform.value,
  })
  settingsRequests.subscribe(settingsPage.configured.openSettings)
  windowTabs.groups.registerKinds(settingsPage.kinds)

  const windowCommands = createHostConnection({
    hostPort: plugins.hostPort,
    pluginHost,
    appCommands: features.featureCommands.commands,
    getTarget: features.getTarget,
    items: palettes.sources.items,
    runCommandToEnd: palettes.commandRuns.runCommandToEnd,
    isPaletteOpen: palettes.paletteSwitch.isOpen,
    commandDeps: palettes.commandDeps,
  })
  const bootstrap = createWindowStart({
    foundation,
    parts,
    getTarget: features.getTarget,
    plugins,
    palettes,
  })

  const base = { frame, tabs: parts.tabsPart, plugins: { host: plugins.support } }
  const commands = {
    getTarget: features.getTarget,
    commands: {
      palette: palettes.search,
      commands: palettes.paletteCommands,
      commandDeps: palettes.commandDeps,
      pluginWays: palettes.plugin.pluginWays,
      runCommand: palettes.commandRuns.runCommand,
      runChoice: palettes.commandRuns.runChoice,
    },
  }
  const kinds = {
    findFileIcon: (path: string) => tabOpeners.findFileIcon(path),
    made: createFileMakers({ tabOpeners, windowTabs, writeMessage: foundation.writeMessage }),
  }
  const frameServices = {
    slots: pluginHost.slots,
    windowCommands,
    views: getPluginViews(pluginHost),
    readNoteKind: createNoteKindReader(features.itemLists),
  }
  const { editing, window, vaults } = parts.vault
  return {
    base,
    services: { editing, window, vaults },
    commands,
    kinds,
    frame: frameServices,
    bootstrap,
    tabs: parts.tabs,
  }
}
