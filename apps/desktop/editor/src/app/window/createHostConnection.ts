/**
 * The connection between the plugin host and the commands of the window: what a plugin reads of the
 * commands, and the commands a plugin runs.
 */
import type { Component } from 'vue'
import type { PluginHost } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandDeps, CommandTarget } from '@/shared/model/command'
import {
  createInvocationRunner,
  createVaultOpener,
  type createWindowPalette,
} from '@/features/command-palette'
import { createWindowCommands, getAppCommandIds, getListedCommandIds, type HostPort } from '@/features/plugin-host'
import type { createCommandSources } from './createCommandSources'

/** What the commands a plugin reads and runs are made from. */
export interface HostConnectionDeps {
  readonly hostPort: HostPort
  readonly pluginHost: Pick<PluginHost, 'commands'>
  readonly appCommands: readonly Command[]
  readonly getTarget: () => CommandTarget
  readonly items: ReturnType<typeof createCommandSources>['items']
  readonly palette: Pick<ReturnType<typeof createWindowPalette>, 'paletteSwitch' | 'runs'>
  readonly commandDeps: CommandDeps
  readonly getCommandIcon: (id: string) => Component | undefined
}

/** Connects the port the plugin host reads to the commands of the window, and returns them. */
export function createHostConnection(deps: HostConnectionDeps) {
  const { hostPort, items, getTarget, commandDeps, palette } = deps
  const appIds = getAppCommandIds(deps.appCommands)
  const windowCommands = createWindowCommands({
    getAppIds: () => appIds,
    getPluginIds: () => getListedCommandIds(deps.pluginHost),
    runCommandToEnd: palette.runs.runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget,
  })
  hostPort.connect({
    commands: windowCommands,
    isPaletteOpen: palette.paletteSwitch.isOpen,
    runInvocation: createInvocationRunner(() => commandDeps),
    openVault: createVaultOpener(commandDeps, getTarget),
    getCommandIcon: deps.getCommandIcon,
  })
  return windowCommands
}
